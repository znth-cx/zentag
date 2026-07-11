package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/znth-cx/zentag/core/cover"
	"github.com/znth-cx/zentag/core/ffmpeg"
	"github.com/znth-cx/zentag/core/isbn"
	"github.com/znth-cx/zentag/core/lang"
	"github.com/znth-cx/zentag/core/mediainfo"
	"github.com/znth-cx/zentag/core/metadata"
	"github.com/znth-cx/zentag/core/sources/audnexus"
	"github.com/znth-cx/zentag/core/sources/files"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

// pgup/pgdn hop tabs; handled by pagedForm since huh has no whole-group
// jump (its Prev/Next only cross at a field boundary).
var (
	pageUpKey   = key.NewBinding(key.WithKeys("pgup"))
	pageDownKey = key.NewBinding(key.WithKeys("pgdown"))
	tabKey      = key.NewBinding(key.WithKeys("tab"))
	shiftTabKey = key.NewBinding(key.WithKeys("shift+tab"))
	quitKey     = key.NewBinding(key.WithKeys("ctrl+c"))
)

// fieldBounds: first/last focusable field keys; a huh.Note has no key
// and is skipped.
func fieldBounds(fields []huh.Field) (first, last string) {
	for _, f := range fields {
		if k := f.GetKey(); k != "" {
			if first == "" {
				first = k
			}
			last = k
		}
	}
	return first, last
}

func runSpinner(ctx context.Context, out io.Writer, title string, action func(context.Context) error) error {
	var actErr error
	spinErr := spinner.New().
		Type(spinner.Dots).
		Title(" " + title).
		Output(out).
		Context(ctx).
		ActionWithErr(func(c context.Context) error {
			actErr = action(c)
			return actErr
		}).
		Run()
	if spinErr != nil {
		return spinErr
	}
	return actErr
}

func init() {
	// huh's field constructors copy the package DefaultKeyMap at build
	// time, so mutating it here (before any form exists) is the only way
	// to reach textinput/textarea's word-motion keys via huh's public API.
	textinput.DefaultKeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w", "ctrl+backspace"))

	textarea.DefaultKeyMap.WordForward = key.NewBinding(key.WithKeys("alt+right", "alt+f", "ctrl+right"))
	textarea.DefaultKeyMap.WordBackward = key.NewBinding(key.WithKeys("alt+left", "alt+b", "ctrl+left"))
	textarea.DefaultKeyMap.DeleteWordForward = key.NewBinding(key.WithKeys("alt+delete", "alt+d", "ctrl+delete"))
	textarea.DefaultKeyMap.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w", "ctrl+backspace"))
}

// pagedForm embeds a multi-group huh.Form in a bubbletea model: pgup/pgdn
// jump a whole tab via NextGroup/PrevGroup, since huh itself only
// advances groups field-by-field. Current group is read from the
// focused field's key (prefixed "g<index>_"); jumps are gated by huh's
// own validation (refuses while the current group has a field error).
//
// onLeave maps a field key to a callback fired once focus leaves it, to
// normalize its value on exit.
//
// actions are extra keybindings (e.g. ctrl+r) that fire fn instead of
// forwarding to the focused field, when its key is in fields (empty
// fields means any field).
type pagedForm struct {
	form     *huh.Form
	lastIdx  int // index of the final group
	curGroup int // last known current group, updated each Update
	onLeave  map[string]func()
	actions  []formAction
	prevKey  string            // focused field key at the end of the previous Update
	bounds   map[int][2]string // group index -> [firstFieldKey, lastFieldKey]

	// files, when non-nil, prepends a Files tab before huh's group 0,
	// reached via pgup from there and pgdn back. See filesTabConfig.
	files    *filesTabConfig
	onFiles  bool // true while the Files tab (list or a dump) is active
	fileIdx  int  // selected index into files.paths
	dumpMode byte // 0 = list view; 'm' or 'f' = showing that dump
	vp       viewport.Model

	// asin, when non-nil, replaces huh's group 0 with a full-screen tab
	// (like Files): an ASIN input plus scrollable search preview. See
	// asinTabConfig.
	asin *asinTabConfig
}

// asinTabConfig enables pagedForm's full-screen ASIN tab: a real
// huh.Input (matches every other tab's theme) plus a scrollable search
// preview. Drives pagedForm directly instead of a huh.Group, since the
// preview needs to fill/scroll the screen like a Files dump, which
// huh's group layout has no room for. Enter confirms and ends the form;
// ctrl+s searches and focuses the preview so ↑/↓ or j/k scroll it.
type asinTabConfig struct {
	field    *huh.Input
	value    string // bound to field via Value(&value); kept in sync on every field.Update
	search   func(asin string) (prettyJSON string, err error)
	browsing bool           // false = editing the input; true = scrolling the preview
	preview  string         // current preview text: instructions, an error, or pretty JSON
	asin     string         // set on confirm (Enter), trimmed; "" means skip the lookup
	vp       viewport.Model // own viewport: pagedForm.vp is Files tab's, sharing clobbers last-rendered tab
}

// asinResultTheme styles the ASIN/Files tabs' headers/dumps to match
// huh's own Note field: NoteTitle for the header, Card for the box,
// whose left border only shows while focused/browsing.
var asinResultTheme = huh.ThemeCharm()

// filesTabConfig enables pagedForm's extra Files tab: a navigable path
// list where 'm'/'f' shows that file's mediainfo/ffprobe dump (esc
// returns to list). fetchDump takes 'm'/'f' and one of paths.
// dumpCache lives here (not on pagedForm) so every TUI stage sharing
// this config reuses dumps instead of re-running subprocesses.
type filesTabConfig struct {
	paths     []string
	fetchDump func(mode byte, path string) (string, error)
	dumpCache map[string]string // "mode\x00path" -> dump text, since source files never change mid-session
}

// newFilesTabConfig builds the Files tab shared by every TUI stage
// (ASIN, conflicts, edit form): one entry per track's source path.
func newFilesTabConfig(ctx context.Context, fw *ffmpeg.Wrapper, mi *mediainfo.Wrapper, tracks []metadata.Track) *filesTabConfig {
	paths := make([]string, len(tracks))
	for i, t := range tracks {
		paths[i] = t.Path
	}
	return &filesTabConfig{
		paths: paths,
		fetchDump: func(mode byte, path string) (string, error) {
			if mode == 'f' {
				return fw.ProbeDump(ctx, path)
			}
			return mi.Dump(ctx, path)
		},
		dumpCache: map[string]string{},
	}
}

// formAction: a keybinding scoped to field keys, runs fn instead of
// forwarding to the focused field. See pagedForm.actions.
type formAction struct {
	binding key.Binding
	fields  map[string]bool // active field keys; empty/nil means any field
	fn      func()
}

func (m *pagedForm) Init() tea.Cmd { return m.form.Init() }

// groupIndexFromKey parses field key prefix "g<digits>" into a group
// index, or -1 if absent.
func groupIndexFromKey(k string) int {
	if !strings.HasPrefix(k, "g") {
		return -1
	}
	i := 1
	for i < len(k) && k[i] >= '0' && k[i] <= '9' {
		i++
	}
	n, err := strconv.Atoi(k[1:i])
	if err != nil {
		return -1
	}
	return n
}

func (m *pagedForm) focusedKey() string {
	if focused := m.form.GetFocusedField(); focused != nil {
		return focused.GetKey()
	}
	return ""
}

// sync refreshes curGroup and fires onLeave for the field just left.
// LANDMINE: must also run right after a NextGroup/PrevGroup jump, else
// a stale curGroup lets rapid pgdn presses over-advance past the last
// tab and submit by accident.
func (m *pagedForm) sync() {
	cur := m.focusedKey()
	if gi := groupIndexFromKey(cur); gi >= 0 {
		m.curGroup = gi
	}
	if m.prevKey != "" && cur != m.prevKey {
		if fn := m.onLeave[m.prevKey]; fn != nil {
			fn()
		}
	}
	m.prevKey = cur
}

func (m *pagedForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		if m.files != nil {
			m.vp.Width = wsz.Width
			m.vp.Height = max(wsz.Height-filesTabChromeHeight, 1)
		}
		if m.asin != nil {
			m.asin.field.WithWidth(wsz.Width)
			m.asin.vp.Width = wsz.Width
			m.asin.vp.Height = max(wsz.Height-filesTabChromeHeight, 1)
		}
	}

	var cmd tea.Cmd
	handled := false
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, pageDownKey):
			if m.onFiles {
				m.onFiles = false
			} else if m.curGroup < m.lastIdx { // never let pgdn on the last tab submit
				cmd = m.form.NextGroup()
			}
			handled = true
		case key.Matches(km, pageUpKey):
			if m.files != nil && !m.onFiles && m.curGroup == 0 {
				m.onFiles = true
			} else if m.curGroup > 0 {
				cmd = m.form.PrevGroup()
			}
			handled = true
		case key.Matches(km, quitKey):
			// handled here since keys never reach m.form.Update while
			// onFiles is true, so huh's own ctrl+c binding wouldn't fire.
			m.form.State = huh.StateAborted
			cmd = m.form.CancelCmd
			handled = true
		case m.onFiles:
			m.handleFilesKey(km)
			handled = true
		case m.asin != nil:
			cmd = m.handleASINKey(km)
			handled = true
		case key.Matches(km, tabKey):
			// swallow tab on a tab's last field: pgdn is the only advance now
			if b, ok := m.bounds[m.curGroup]; ok && m.curGroup < m.lastIdx && m.focusedKey() == b[1] {
				handled = true
			}
		case key.Matches(km, shiftTabKey):
			// mirror: swallow shift+tab on a tab's first field
			if b, ok := m.bounds[m.curGroup]; ok && m.curGroup > 0 && m.focusedKey() == b[0] {
				handled = true
			}
		default:
			cur := m.focusedKey()
			for _, a := range m.actions {
				if key.Matches(km, a.binding) && (len(a.fields) == 0 || a.fields[cur]) {
					a.fn()
					handled = true
					break
				}
			}
		}
	}

	if !handled {
		model, c := m.form.Update(msg)
		if f, ok := model.(*huh.Form); ok {
			m.form = f
		}
		cmd = c
	}

	m.sync()
	if m.form.State != huh.StateNormal {
		return m, tea.Quit
	}
	return m, cmd
}

func (m *pagedForm) View() string {
	if m.form.State != huh.StateNormal {
		return ""
	}
	if m.onFiles {
		return m.viewFiles()
	}
	if m.asin != nil {
		return m.viewASIN()
	}
	return m.form.View()
}

// filesTabChromeHeight: rows viewFiles reserves for header/footer
// around the viewport when a dump shows.
const filesTabChromeHeight = 4

var (
	fileListSelectedStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	fileListHintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// handleFilesKey handles a key while the Files tab is active. pgup/pgdn
// are intercepted in Update, so they always leave/re-enter the tab.
func (m *pagedForm) handleFilesKey(km tea.KeyMsg) {
	if m.dumpMode != 0 {
		switch km.String() {
		case "esc":
			m.dumpMode = 0
		case "up", "k":
			m.vp.ScrollUp(1)
		case "down", "j":
			m.vp.ScrollDown(1)
		}
		return
	}

	switch km.String() {
	case "up", "k":
		if m.fileIdx > 0 {
			m.fileIdx--
		}
	case "down", "j":
		if m.fileIdx < len(m.files.paths)-1 {
			m.fileIdx++
		}
	case "m":
		m.openDump('m')
	case "f":
		m.openDump('f')
	}
}

// openDump fetches (or reuses cached) mediainfo/ffprobe dump for mode
// ('m'/'f'). Source files never change mid-session, so cache forever.
func (m *pagedForm) openDump(mode byte) {
	path := m.files.paths[m.fileIdx]
	cacheKey := string(mode) + "\x00" + path

	text, ok := m.files.dumpCache[cacheKey]
	if !ok {
		var err error
		text, err = m.files.fetchDump(mode, path)
		if err != nil {
			m.dumpMode = mode
			m.vp.SetContent(err.Error())
			m.vp.GotoTop()
			return
		}
		m.files.dumpCache[cacheKey] = text
	}

	m.dumpMode = mode
	m.vp.SetContent(text)
	m.vp.GotoTop()
}

// viewFiles renders the path list, or a dump once 'm'/'f' was pressed.
func (m *pagedForm) viewFiles() string {
	if m.dumpMode != 0 {
		styles := &asinResultTheme.Focused
		tool := "mediainfo"
		if m.dumpMode == 'f' {
			tool = "ffprobe"
		}
		header := styles.NoteTitle.Render(fmt.Sprintf("%s — %s", tool, m.files.paths[m.fileIdx]))
		hint := fileListHintStyle.Render("esc back to list · ↑/↓ scroll · pgup/pgdn switch tabs")
		return header + "\n\n" + styles.Card.Render(m.vp.View()) + "\n" + hint
	}

	// list also reuses vp: offset clamped to keep selection visible so a
	// long list scrolls instead of getting cut off
	lines := make([]string, len(m.files.paths))
	for i, p := range m.files.paths {
		if i == m.fileIdx {
			lines[i] = fileListSelectedStyle.Render(p)
		} else {
			lines[i] = p
		}
	}
	m.vp.SetContent(strings.Join(lines, "\n"))
	switch {
	case m.fileIdx < m.vp.YOffset:
		m.vp.SetYOffset(m.fileIdx)
	case m.fileIdx >= m.vp.YOffset+m.vp.Height:
		m.vp.SetYOffset(m.fileIdx - m.vp.Height + 1)
	}

	styles := &asinResultTheme.Blurred
	header := styles.NoteTitle.Render("Files")
	hint := fileListHintStyle.Render("↑/↓ select · m mediainfo dump · f ffprobe dump · pgdn next tab")
	return header + "\n\n" + styles.Card.Render(m.vp.View()) + "\n" + hint
}

// handleASINKey: pgup/pgdn/ctrl+c are intercepted in Update, so they
// always switch tabs or abort. Unhandled keys forward to the ASIN
// field so normal text editing keeps working.
func (m *pagedForm) handleASINKey(km tea.KeyMsg) tea.Cmd {
	a := m.asin
	switch {
	case key.Matches(km, key.NewBinding(key.WithKeys("ctrl+s"))):
		m.searchASIN()
		return nil
	case km.String() == "enter":
		a.asin = strings.TrimSpace(a.value)
		m.form.State = huh.StateCompleted
		return m.form.SubmitCmd
	case a.browsing:
		switch km.String() {
		case "esc":
			a.browsing = false
			return a.field.Focus()
		case "up", "k":
			a.vp.ScrollUp(1)
		case "down", "j":
			a.vp.ScrollDown(1)
		}
		return nil
	default:
		model, cmd := a.field.Update(km)
		if f, ok := model.(*huh.Input); ok {
			a.field = f
		}
		return cmd
	}
}

// searchASIN fetches+stores the preview, then focuses it so ↑/↓ or j/k
// scroll immediately, no second keypress needed.
func (m *pagedForm) searchASIN() {
	a := m.asin
	val := strings.TrimSpace(a.value)
	if val == "" {
		a.preview = "enter an ASIN, then ctrl+s to search"
		a.browsing = false
		return
	}
	text, err := a.search(val)
	if err != nil {
		a.preview = fmt.Sprintf("search failed: %v", err)
	} else {
		a.preview = text
	}
	a.browsing = true
	a.field.Blur()
	a.vp.GotoTop()
}

// viewASIN renders the themed ASIN input plus a scrollable preview of
// the last search result (or instructions/error), styled like a huh
// Note; see asinResultTheme.
func (m *pagedForm) viewASIN() string {
	a := m.asin
	styles := &asinResultTheme.Blurred
	header := "audnexus result"
	if a.browsing {
		styles = &asinResultTheme.Focused
		header += " — esc to edit ASIN"
	}
	a.vp.SetContent(a.preview)
	result := styles.Card.Render(styles.NoteTitle.Render(header) + "\n" + a.vp.View())

	hint := fileListHintStyle.Render("ctrl+s search · ↑/↓ or j/k scroll result · enter confirm & continue · pgup Files tab")
	return a.field.View() + "\n" + result + "\n" + hint
}

// runPagedForm runs form through bubbletea, enabling pgup/pgdn tab
// hopping (lastIdx = final group index). bounds maps a group index to
// its [first, last] field keys, to swallow tab/shift+tab at a tab's
// edge; nil (or a missing group) leaves tab/shift+tab untouched. files
// prepends the Files tab (pgup from group 0); asin replaces group 0
// with the full-screen ASIN tab. No caller uses both at once, though
// nothing enforces that. Returns the form's terminal State.
func runPagedForm(ctx context.Context, in io.Reader, out io.Writer, form *huh.Form, lastIdx int, onLeave map[string]func(), actions []formAction, bounds map[int][2]string, files *filesTabConfig, asin *asinTabConfig) (huh.FormState, error) {
	form.SubmitCmd = tea.Quit
	form.CancelCmd = tea.Interrupt
	m := &pagedForm{form: form, lastIdx: lastIdx, onLeave: onLeave, actions: actions, bounds: bounds, files: files, asin: asin}

	final, err := tea.NewProgram(m,
		tea.WithContext(ctx),
		tea.WithInput(in),
		tea.WithOutput(out),
	).Run()
	if err != nil && !errors.Is(err, tea.ErrInterrupted) {
		return huh.StateAborted, err
	}
	return final.(*pagedForm).form.State, nil
}

// runConflictForm: interactive phase-1 replacement for promptConflicts.
// One huh Select per conflict (candidates labeled by origin, plus
// "omit"), defaulted to Recommended, alongside the Files tab. Returns
// the same field->choice map as promptConflicts. accepted is false on
// abort (nil error), but choices still reflect selections so far so the
// caller can persist progress before cancelling.
func runConflictForm(ctx context.Context, in io.Reader, out io.Writer, conflicts []metadata.Conflict, files *filesTabConfig) (choices map[string]int, accepted bool, err error) {
	picks := make([]int, len(conflicts))
	fields := make([]huh.Field, 0, len(conflicts))

	for i, c := range conflicts {
		picks[i] = c.Recommended

		opts := make([]huh.Option[int], 0, len(c.Values)+1)
		for j, v := range c.Values {
			label := fmt.Sprintf("%s  [%s]", v, c.Origins[j])
			if j == c.Recommended {
				label += "  (recommended)"
			}
			opts = append(opts, huh.NewOption(label, j))
		}
		opts = append(opts, huh.NewOption("(omit this field)", -1))

		fields = append(fields, huh.NewSelect[int]().Key(fmt.Sprintf("g0_%d", i)).
			Title(c.Field).
			Description("sources disagree — pick the value to keep").
			Options(opts...).
			Value(&picks[i]))
	}

	groupBounds := map[int][2]string{}
	first, last := fieldBounds(fields)
	groupBounds[0] = [2]string{first, last}

	form := huh.NewForm(huh.NewGroup(fields...).
		Title("Resolve conflicts").
		Description("pgup/pgdn switch tabs"))

	state, runErr := runPagedForm(ctx, in, out, form, 0, nil, nil, groupBounds, files, nil)
	if runErr != nil {
		return nil, false, runErr
	}
	aborted := state == huh.StateAborted

	// build the map from current selections regardless of submit/abort
	resolved := make(map[string]int, len(conflicts))
	for i, c := range conflicts {
		resolved[c.Field] = picks[i]
	}
	return resolved, !aborted, nil
}

// runASINForm: interactive replacement for a plain ASIN prompt. Always
// runs when interactive (unless --asin passed), prefilled with initial
// (resolveASIN's file-tag value, or ""), as a full-screen tab alongside
// Files (pgup); Book data/Chapters/Submit don't exist yet, nothing's
// merged. ctrl+s pretty-prints audnexus's raw /books/{ASIN} JSON for
// preview. Enter confirms and, if non-blank, runs the real
// audnexus.Gather once regardless of ctrl+s. Blank ASIN or abort both
// skip the lookup like a failed lookup would; neither blocks the run.
func runASINForm(ctx context.Context, in io.Reader, out io.Writer, w *audnexus.Wrapper, region, initial string, files *filesTabConfig) (asin string, apiMeta *metadata.Metadata, err error) {
	asinCfg := &asinTabConfig{
		value:   initial,
		preview: "enter an ASIN, then ctrl+s to search",
		search: func(a string) (string, error) {
			raw, fetchErr := audnexus.FetchBookJSON(ctx, w, a, region)
			if fetchErr != nil {
				return "", fetchErr
			}
			var pretty bytes.Buffer
			if jsonErr := json.Indent(&pretty, raw, "", "  "); jsonErr != nil {
				return string(raw), nil
			}
			return pretty.String(), nil
		},
	}
	asinCfg.field = huh.NewInput().
		Title("ASIN").
		Description("Audible ASIN for the audnexus lookup — leave blank to skip · ctrl+s to search · enter to confirm & continue").
		Placeholder("e.g. B08G9PRS1K").
		Value(&asinCfg.value)
	asinCfg.field.Init() // huh.Input.Init() blurs by default
	asinCfg.field.Focus()

	// bare huh.Form only to drive pagedForm's completion state machine
	// (handleASINKey sets State directly); its field is never rendered,
	// viewASIN fully replaces huh's view for this tab.
	form := huh.NewForm(huh.NewGroup(huh.NewNote()))

	state, runErr := runPagedForm(ctx, in, out, form, 0, nil, nil, nil, files, asinCfg)
	if runErr != nil {
		return "", nil, runErr
	}
	if state == huh.StateAborted || asinCfg.asin == "" {
		return "", nil, nil
	}

	meta, gatherErr := audnexus.Gather(ctx, w, asinCfg.asin, region)
	if gatherErr != nil {
		logger.Warn("transform: audnexus lookup failed, continuing without it", "asin", asinCfg.asin, "error", gatherErr)
		return asinCfg.asin, nil, nil
	}
	return asinCfg.asin, meta, nil
}

// promptPartNumberRegex asks for a Go regexp (first capture group =
// part number) when GatherGreedy's default leading-digits pattern
// doesn't match. ctrl+r previews the filename->part mapping via
// files.PreviewPartNumbers; the field's Validate hook runs the same
// check and blocks submit until every file parses. Abort returns
// (nil, nil) so the caller falls back to the original gather error.
func promptPartNumberRegex(ctx context.Context, in io.Reader, out io.Writer, path string) (*regexp.Regexp, error) {
	var pattern string

	preview := huh.NewNote().
		Title("Preview").
		Description("ctrl+r to preview the filename → part number mapping")

	renderPreview := func() {
		re, err := regexp.Compile(pattern)
		switch {
		case pattern == "":
			preview.Description("enter a pattern, then ctrl+r to preview")
		case err != nil:
			preview.Description(fmt.Sprintf("invalid regexp: %v", err))
		case re.NumSubexp() == 0:
			preview.Description("pattern needs a capture group for the part number")
		default:
			results, err := files.PreviewPartNumbers(path, re)
			if err != nil {
				preview.Description(err.Error())
				return
			}
			var b strings.Builder
			for _, r := range results {
				if r.Err != nil {
					fmt.Fprintf(&b, "%s -> ERROR: %v\n", r.Filename, r.Err)
				} else {
					fmt.Fprintf(&b, "%s -> %d\n", r.Filename, r.Part)
				}
			}
			preview.Description(strings.TrimRight(b.String(), "\n"))
		}
	}

	validatePattern := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("enter a pattern")
		}
		re, err := regexp.Compile(s)
		if err != nil {
			return fmt.Errorf("invalid regexp: %v", err)
		}
		if re.NumSubexp() == 0 {
			return fmt.Errorf("pattern needs a capture group for the part number")
		}
		results, err := files.PreviewPartNumbers(path, re)
		if err != nil {
			return err
		}
		for _, r := range results {
			if r.Err != nil {
				return fmt.Errorf("%q: %v", r.Filename, r.Err)
			}
		}
		return nil
	}

	patternField := huh.NewInput().Key("g0_part_number_regex").
		Title("Part number regex").
		Description(`Go regexp with a capture group for the part number, e.g. ^Track (\d+) · ctrl+r preview`).
		Validate(validatePattern).
		Value(&pattern)

	form := huh.NewForm(huh.NewGroup(
		huh.NewNote().
			Title("No part number found").
			Description(fmt.Sprintf("%q has multiple files but none match the default leading-digit pattern.", path)),
		patternField,
		preview,
	)).WithInput(in).WithOutput(out)

	actions := []formAction{
		{binding: key.NewBinding(key.WithKeys("ctrl+r")), fields: map[string]bool{"g0_part_number_regex": true}, fn: renderPreview},
	}
	state, err := runPagedForm(ctx, in, out, form, 0, nil, actions, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if state == huh.StateAborted {
		return nil, nil
	}

	re, err := regexp.Compile(pattern) // re-derived; already passed validatePattern
	if err != nil {
		return nil, fmt.Errorf("transform: part number regex: %w", err)
	}
	return re, nil
}

// chapRef locates one chapter so an edited title (flat titles slice,
// same index) writes back to the right track/chapter.
type chapRef struct{ track, chapter int }

// runEditForm: interactive phase-2 replacement for confirmProceed, a
// three-tab huh form over a copy of m (Book data / Chapters / Submit).
// Always returns the edited copy; accepted is true only on Submit
// confirm, false on abort or decline (caller persists the copy either
// way so nothing typed is lost). m itself is never mutated.
// filesCfg: shared Files tab config (reuses dump cache across stages);
// nil builds a fresh one.
func runEditForm(ctx context.Context, in io.Reader, out io.Writer, m *metadata.Metadata, fw *ffmpeg.Wrapper, mi *mediainfo.Wrapper, filesCfg *filesTabConfig) (edited *metadata.Metadata, accepted bool, err error) {
	title := m.Title
	subtitle := m.Subtitle
	author := strings.Join(m.Author, ", ")
	publisher := strings.Join(m.Publisher, ", ")
	narrator := strings.Join(m.Narrator, ", ")
	genre := strings.Join(m.Genre, ", ")
	description := m.Description
	language := m.Language
	isbn := m.ISBN
	asin := m.ASIN
	edition := m.Edition
	year := ""
	if m.Year != 0 {
		year = strconv.Itoa(m.Year)
	}
	var seriesName, seriesPart string
	if len(m.Series) > 0 {
		seriesName = m.Series[0].Name
		seriesPart = m.Series[0].Part
	}
	source := m.Source
	coverPath := ""

	// flat titles aligned by index with refs; computed first so tab
	// count/layout is known before building any group
	var titles []string
	var refs []chapRef
	for ti := range m.Tracks {
		for ci := range m.Tracks[ti].Chapters {
			titles = append(titles, m.Tracks[ti].Chapters[ci].Title)
			refs = append(refs, chapRef{ti, ci})
		}
	}
	hasChapters := len(refs) > 0

	// Chapters only gets a tab when the item has chapters, so Submit is
	// index 1 or 2. Field keys carry "g<idx>_" so pagedForm can tell
	// which tab is focused.
	bookIdx, chapIdx, submitIdx := 0, 1, 1
	total := 2
	if hasChapters {
		submitIdx, total = 2, 3
	}
	// dispOffset accounts for the Files tab (not a huh group, so not
	// counted in bookIdx/chapIdx/submitIdx/total above).
	const dispOffset = 1
	const navHint = "pgup/pgdn switch tabs · alt+←/→ word · ctrl+w delete word"

	// explicit accessor so onLeave can normalize a typed name to its ISO
	// code and push it back into the field display
	langAcc := huh.NewPointerAccessor(&language)
	langField := huh.NewInput().Key("g0_language").
		Title("Language").
		Description(`ISO-639-3 code, e.g. eng; a name like "English" is converted on exit`).
		Validate(validateLanguage).
		Accessor(langAcc)

	bookFields := []huh.Field{
		huh.NewInput().Key("g0_title").Title("Title").Value(&title),
		huh.NewInput().Key("g0_subtitle").Title("Subtitle").Value(&subtitle),
		huh.NewInput().Key("g0_author").Title("Author").Description("comma-separated, primary first").Value(&author),
		huh.NewInput().Key("g0_publisher").Title("Publisher").Description("comma-separated, primary first").Value(&publisher),
		huh.NewInput().Key("g0_year").Title("Year").Validate(validateYear).Value(&year),
		huh.NewInput().Key("g0_narrator").Title("Narrator").Description("comma-separated, primary first").Value(&narrator),
		huh.NewInput().Key("g0_genre").Title("Genre").Description("comma-separated, primary first").Value(&genre),
		huh.NewInput().Key("g0_series_name").Title("Series name").Value(&seriesName),
		huh.NewInput().Key("g0_series_part").Title("Series part").Value(&seriesPart),
		langField,
		huh.NewInput().Key("g0_isbn").Title("ISBN").Validate(validateISBN).Value(&isbn),
		huh.NewInput().Key("g0_asin").Title("ASIN").Value(&asin),
		huh.NewInput().Key("g0_edition").Title("Edition").Value(&edition),
		huh.NewSelect[metadata.ReleaseSource]().Key("g0_source").Title("Source").Options(
			huh.NewOption("WEB", metadata.ReleaseSourceWEB),
			huh.NewOption("CD", metadata.ReleaseSourceCD),
			huh.NewOption("VINYL", metadata.ReleaseSourceVinyl),
			huh.NewOption("CASSETTE", metadata.ReleaseSourceCassette),
		).Value(&source),
		huh.NewText().Key("g0_description").Title("Description").Value(&description),
		huh.NewInput().Key("g0_cover").Title("Cover").Description(coverStatus(m)).
			Placeholder("leave blank to keep; or enter a new filepath/URL").Value(&coverPath),
	}

	// groupBounds: tab index -> [first, last] field keys, so pagedForm
	// can swallow tab/shift+tab at a tab's edge.
	groupBounds := map[int][2]string{}
	first, last := fieldBounds(bookFields)
	groupBounds[bookIdx] = [2]string{first, last}

	groups := []*huh.Group{
		huh.NewGroup(bookFields...).
			Title("Book data").Description(fmt.Sprintf("Tab %d of %d · %s", bookIdx+1+dispOffset, total+dispOffset, navHint)),
	}

	var regexActions []formAction // nil when item has no chapters

	if hasChapters {
		// LANDMINE: mutating the bound string alone doesn't refresh an
		// already-built Input's internal textinput buffer, so chapInputs
		// mirrors the *huh.Input values for the bulk-replace to push into.
		chapInputs := make([]*huh.Input, len(refs))
		chapFields := make([]huh.Field, 0, len(refs))
		multiTrack := len(m.Tracks) > 1
		for k, ref := range refs {
			ch := m.Tracks[ref.track].Chapters[ref.chapter]
			label := fmt.Sprintf("Chapter %d", ref.chapter+1)
			if multiTrack {
				label = fmt.Sprintf("Track %d · Chapter %d", ref.track+1, ref.chapter+1)
			}
			f := huh.NewInput().
				Key(fmt.Sprintf("g1_%d", k)).
				Title(label).
				Description(fmt.Sprintf("%s – %s", formatDuration(ch.Start), formatDuration(ch.End))).
				Value(&titles[k])
			chapInputs[k] = f
			chapFields = append(chapFields, f)
		}
		refreshChapterFields := func() {
			for k := range chapInputs {
				chapInputs[k].Accessor(huh.NewPointerAccessor(&titles[k]))
			}
		}

		// Bulk rename: ctrl+r applies regex to every title, ctrl+z undoes
		// last replace. Both no-op silently (no room to surface an error
		// from a keybinding) rather than erroring.
		var regexSearch, regexReplace string
		var preReplace []string // snapshot from just before the last replace; nil if none yet
		applyRegex := func() {
			re, err := regexp.Compile(regexSearch)
			if regexSearch == "" || err != nil {
				return
			}
			prev := make([]string, len(titles))
			copy(prev, titles)
			for i := range titles {
				titles[i] = re.ReplaceAllString(titles[i], regexReplace)
			}
			preReplace = prev
			refreshChapterFields()
		}
		resetRegex := func() {
			if preReplace == nil {
				return
			}
			copy(titles, preReplace)
			preReplace = nil
			refreshChapterFields()
		}
		regexActions = []formAction{
			{binding: key.NewBinding(key.WithKeys("ctrl+r")), fields: map[string]bool{"g1_regex_search": true, "g1_regex_replace": true}, fn: applyRegex},
			{binding: key.NewBinding(key.WithKeys("ctrl+z")), fields: map[string]bool{"g1_regex_search": true, "g1_regex_replace": true}, fn: resetRegex},
		}

		regexFields := []huh.Field{
			huh.NewInput().Key("g1_regex_search").
				Title("Regex Search").
				Description("Go regexp · ctrl+r applies to every chapter title below · ctrl+z undoes last replace").
				Validate(validateRegexPattern).
				Value(&regexSearch),
			huh.NewInput().Key("g1_regex_replace").
				Title("Regex Replace").
				Description("replacement text, may use $1 etc. for capture groups · ctrl+r apply · ctrl+z undo").
				Value(&regexReplace),
		}
		chapFields = append(regexFields, chapFields...)
		first, last := fieldBounds(chapFields)
		groupBounds[chapIdx] = [2]string{first, last}

		groups = append(groups, huh.NewGroup(chapFields...).
			Title("Chapters").
			Description(fmt.Sprintf("Tab %d of %d · edit chapter titles · times read-only · %s", chapIdx+1+dispOffset, total+dispOffset, navHint)))
	}

	confirmWrite := true
	confirmKey := fmt.Sprintf("g%d_confirm", submitIdx)
	groupBounds[submitIdx] = [2]string{confirmKey, confirmKey}
	groups = append(groups, huh.NewGroup(
		huh.NewNote().
			Title("Submit").
			Description("You are responsible for the accuracy of every field.\n"+
				"pgup to review the other tabs before writing."),
		huh.NewConfirm().
			Key(confirmKey).
			Title("Write these files?").
			Affirmative("Yes, write").
			Negative("No, cancel").
			Value(&confirmWrite),
	).Title("Submit").Description(fmt.Sprintf("Tab %d of %d", submitIdx+1+dispOffset, total+dispOffset)))

	// normalize on exit, not just at save time
	onLeave := map[string]func(){
		"g0_language": func() {
			if code, ok := resolveLanguage(language); ok && code != language {
				language = code
				langField.Accessor(langAcc) // push normalized value into the field display
			}
		},
	}

	// paths come from already-gathered metadata, not a re-scan/filename parse
	if filesCfg == nil {
		filesCfg = newFilesTabConfig(ctx, fw, mi, m.Tracks)
	}

	form := huh.NewForm(groups...).WithShowHelp(true)
	state, runErr := runPagedForm(ctx, in, out, form, submitIdx, onLeave, regexActions, groupBounds, filesCfg, nil)
	if runErr != nil {
		return nil, false, runErr
	}
	aborted := state == huh.StateAborted

	// applies onto a copy; m never mutated; runs even on abort so
	// in-progress edits can be saved to the session
	out2 := *m
	out2.Title = strings.TrimSpace(title)
	out2.Subtitle = strings.TrimSpace(subtitle)
	out2.Author = splitCSV(author)
	out2.Publisher = splitCSV(publisher)
	out2.Narrator = splitCSV(narrator)
	out2.Genre = splitCSV(genre)
	out2.Description = strings.TrimSpace(description)
	if code, ok := resolveLanguage(language); ok {
		out2.Language = code // normalize "English"/"ENG" -> "eng"
	} else {
		out2.Language = strings.TrimSpace(language)
	}
	out2.ISBN = strings.TrimSpace(isbn)
	out2.ASIN = strings.TrimSpace(asin)
	out2.Edition = strings.TrimSpace(edition)
	out2.Source = source

	out2.Year = 0
	if y := strings.TrimSpace(year); y != "" {
		out2.Year, _ = strconv.Atoi(y) // validated by validateYear
	}

	if seriesName = strings.TrimSpace(seriesName); seriesName != "" {
		out2.Series = []metadata.SeriesEntry{{Name: seriesName, Part: strings.TrimSpace(seriesPart)}}
	} else {
		out2.Series = nil
	}

	if cp := strings.TrimSpace(coverPath); cp != "" {
		img, mime, loadErr := cover.Load(ctx, cp)
		switch {
		case loadErr == nil:
			out2.CoverImage = img
			out2.CoverMIME = mime
		case aborted:
			// abort with a half-typed cover shouldn't fail the save: keep
			// existing cover, drop the bad path
		default:
			return nil, false, fmt.Errorf("transform: load cover %q: %w", cp, loadErr)
		}
	}

	// Deep-copy tracks before writing edited chapter titles; input m's slices must stay untouched.
	out2.Tracks = make([]metadata.Track, len(m.Tracks))
	copy(out2.Tracks, m.Tracks)
	for ti := range out2.Tracks {
		chs := make([]metadata.Chapter, len(m.Tracks[ti].Chapters))
		copy(chs, m.Tracks[ti].Chapters)
		out2.Tracks[ti].Chapters = chs
	}
	for k, ref := range refs {
		out2.Tracks[ref.track].Chapters[ref.chapter].Title = strings.TrimSpace(titles[k])
	}

	return &out2, !aborted && confirmWrite, nil
}

// validateYear accepts blank (unset) or a positive integer.
func validateYear(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fmt.Errorf("year must be a positive number")
	}
	return nil
}

// validateISBN accepts blank (unset) or a checksum-valid ISBN.
func validateISBN(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	ok, err := isbn.Validate(s)
	if err != nil {
		return fmt.Errorf("ISBN must be 10 or 13 digits")
	}
	if !ok {
		return fmt.Errorf("ISBN checksum is invalid")
	}
	return nil
}

// validateRegexPattern accepts blank (unset) or a compilable regexp.
func validateRegexPattern(s string) error {
	if s == "" {
		return nil
	}
	if _, err := regexp.Compile(s); err != nil {
		return fmt.Errorf("invalid regexp: %v", err)
	}
	return nil
}

// resolveLanguage normalizes s to a lowercase ISO-639-3 code: accepts a
// code (any case), an English name ("English" -> "eng"), or "en".
func resolveLanguage(s string) (code string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return lang.ResolveNameOrCode(s)
}

// validateLanguage accepts blank (unset) or anything resolveLanguage resolves.
func validateLanguage(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if _, ok := resolveLanguage(s); !ok {
		return fmt.Errorf("use an ISO-639-3 code (e.g. eng), \"en\", or a language name (e.g. English)")
	}
	return nil
}

func coverStatus(m *metadata.Metadata) string {
	if len(m.CoverImage) > 0 {
		return fmt.Sprintf("current: %s, %d bytes", m.CoverMIME, len(m.CoverImage))
	}
	return "current: none"
}

// formatDuration renders d as H:MM:SS.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d / time.Second)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}
