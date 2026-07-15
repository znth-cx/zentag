package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/znth-cx/zentag/core/ruleset"
)

func TestFormatReport_EmptyText(t *testing.T) {
	got, err := formatReport(nil, false)
	require.NoError(t, err)
	assert.Equal(t, "No rule violations found.", got)
}

func TestFormatReport_EmptyJSON(t *testing.T) {
	got, err := formatReport(nil, true)
	require.NoError(t, err)
	assert.Equal(t, "[]", got)
}

func TestFormatReport_GroupsBySeverityWorstFirst(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "chapters", Severity: ruleset.SeverityTrumpable, Message: "track missing chapters"},
		{Rule: "banned_content", Severity: ruleset.SeverityProhibited, Message: "author banned"},
		{Rule: "bitrate", Severity: ruleset.SeverityUpgradable, Message: "bitrate too low"},
	}
	got, err := formatReport(violations, false)
	require.NoError(t, err)
	want := "PROHIBITED (1)\n" +
		"- [banned_content] author banned\n" +
		"\n" +
		"UPGRADABLE (1)\n" +
		"- [bitrate] bitrate too low\n" +
		"\n" +
		"TRUMPABLE (1)\n" +
		"- [chapters] track missing chapters"
	assert.Equal(t, want, got)
}

func TestFormatReport_SkipsEmptySeverityGroups(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "primary_keys", Severity: ruleset.SeverityTrumpable, Message: "no ISBN or ASIN"},
	}
	got, err := formatReport(violations, false)
	require.NoError(t, err)
	assert.Equal(t, "TRUMPABLE (1)\n- [primary_keys] no ISBN or ASIN", got)
}

func TestFormatCheckReport_PassedFirstFailedLast(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "naming", Severity: ruleset.SeverityTrumpable, Message: "bad directory name"},
		{Rule: "primary_keys", Severity: ruleset.SeverityTrumpable, Message: "no ISBN or ASIN"},
	}
	got := formatCheckReport(violations)
	lines := strings.Split(got, "\n")

	require.Contains(t, got, "✗ primary_keys")
	require.Contains(t, got, "✗ naming")
	require.Contains(t, got, "no ISBN or ASIN")
	require.Contains(t, got, "bad directory name")

	// passing lines must precede failing lines
	firstFail := -1
	lastPass := -1
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "✓"):
			lastPass = i
		case strings.HasPrefix(line, "✗"):
			if firstFail == -1 {
				firstFail = i
			}
		}
	}
	require.NotEqual(t, -1, firstFail)
	require.NotEqual(t, -1, lastPass)
	assert.Less(t, lastPass, firstFail)
}

func TestFormatCheckReport_WarnOnlyRuleRendersYellowTriangle(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "audnexus_chapters", Severity: ruleset.SeverityWarn, Message: "audnexus reports 20 chapters, item has 18"},
	}
	got := formatCheckReport(violations)
	assert.Contains(t, got, "⚠ audnexus_chapters")
	assert.NotContains(t, got, "✗ audnexus_chapters")
}

func TestFormatCheckReport_MixedSeverityRuleStillFails(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "chapters", Severity: ruleset.SeverityWarn, Message: "advisory"},
		{Rule: "chapters", Severity: ruleset.SeverityTrumpable, Message: "real failure"},
	}
	got := formatCheckReport(violations)
	assert.Contains(t, got, "✗ chapters")
	assert.NotContains(t, got, "⚠ chapters")
}

func TestFormatCheckReport_AllPassed(t *testing.T) {
	got := formatCheckReport(nil)
	for _, rule := range checkRuleOrder {
		assert.Contains(t, got, "✓ "+rule)
	}
	assert.NotContains(t, got, "✗")
}

// TestFormatCheckReport_StrayRuleKeyStillRenders: a rule key not in
// checkRuleOrder (e.g. a new check added without updating the list) must
// still appear in output. Regression for the silent-drop bug where check
// printed "violations found" but showed nothing.
func TestFormatCheckReport_StrayRuleKeyStillRenders(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "some_future_rule", Severity: ruleset.SeverityTrumpable, Message: "future check failed"},
	}
	got := formatCheckReport(violations)
	assert.Contains(t, got, "✗ some_future_rule")
	assert.Contains(t, got, "future check failed")
}

// TestFormatCheckReport_StrayWarnOnlyRendersTriangle: stray rules with only
// SeverityWarn render as advisory (⚠), not hard failure (✗).
func TestFormatCheckReport_StrayWarnOnlyRendersTriangle(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "future_advisory", Severity: ruleset.SeverityWarn, Message: "advisory"},
	}
	got := formatCheckReport(violations)
	assert.Contains(t, got, "⚠ future_advisory")
	assert.NotContains(t, got, "✗ future_advisory")
}

// TestCheckRuleOrderMatchesRulesetKeys guards future drift: every Rule key
// the ruleset actually emits must be in checkRuleOrder so the canonical
// display order applies. If this fails, add the missing key to checkRuleOrder
// in report.go (and keep it in Validate call order).
func TestCheckRuleOrderMatchesRulesetKeys(t *testing.T) {
	// Derived from every Rule: "..." literal in core/ruleset/*.go. A new
	// check that adds a Rule literal must also add it here.
	emitted := map[string]struct{}{
		"primary_keys":           {},
		"required_tags":          {},
		"language":               {},
		"cover":                  {},
		"cover_placement":        {},
		"chapters":               {},
		"audnexus_chapters":      {},
		"banned_content":         {},
		"naming":                 {},
		"source":                 {},
		"format_specific_tags":   {},
		"m4b_split_file":         {},
		"extra_files":            {},
		"bitrate":                {},
		"lossy_container":        {},
		"mixed_format":          {},
		"flac_md5":               {},
		"tag_separator_format":   {},
	}
	listed := make(map[string]struct{}, len(checkRuleOrder))
	for _, r := range checkRuleOrder {
		listed[r] = struct{}{}
	}
	for k := range emitted {
		_, ok := listed[k]
		assert.True(t, ok, "rule %q emitted by ruleset but missing from checkRuleOrder in report.go", k)
	}
}

func TestFormatReport_JSONRoundTrips(t *testing.T) {
	violations := []ruleset.Violation{
		{Rule: "primary_keys", Severity: ruleset.SeverityTrumpable, Message: "no ISBN or ASIN"},
	}
	got, err := formatReport(violations, true)
	require.NoError(t, err)

	var decoded []ruleset.Violation
	require.NoError(t, json.Unmarshal([]byte(got), &decoded))
	assert.Equal(t, violations, decoded)
}
