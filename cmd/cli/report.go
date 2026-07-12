package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/znth-cx/zentag/core/ruleset"
)

// severityOrder: display order, worst (Prohibited) first.
var severityOrder = []ruleset.Severity{
	ruleset.SeverityProhibited,
	ruleset.SeverityUpgradable,
	ruleset.SeverityTrumpable,
	ruleset.SeverityWarn,
}

// checkRuleOrder: rule areas ruleset.Validate checks, in check order;
// drives one pass/fail line per area in `zentag check` output.
var checkRuleOrder = []string{
	"primary_keys",
	"required_tags",
	"language",
	"cover",
	"chapters",
	"audnexus_chapters",
	"banned_content",
	"naming",
}

var (
	checkPassStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	checkFailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	checkWarnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	checkMsgStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// allWarn: true if every violation in group is SeverityWarn (yellow
// triangle/advisory) rather than a red x (actual failure).
func allWarn(group []ruleset.Violation) bool {
	for _, v := range group {
		if v.Severity != ruleset.SeverityWarn {
			return false
		}
	}
	return true
}

// formatCheckReport renders one line per rule area: green check, red x,
// or yellow triangle (all-SeverityWarn, advisory not failure). Passing
// areas first, failing/warning areas (with indented messages) last.
func formatCheckReport(violations []ruleset.Violation) string {
	byRule := make(map[string][]ruleset.Violation)
	for _, v := range violations {
		byRule[v.Rule] = append(byRule[v.Rule], v)
	}

	var passed, failed []string
	for _, rule := range checkRuleOrder {
		group, failing := byRule[rule]
		if !failing {
			passed = append(passed, checkPassStyle.Render("✓")+" "+rule)
			continue
		}
		symbol := checkFailStyle.Render("✗")
		if allWarn(group) {
			symbol = checkWarnStyle.Render("⚠")
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%s %s", symbol, rule)
		for _, v := range group {
			fmt.Fprintf(&b, "\n  %s", checkMsgStyle.Render(fmt.Sprintf("[%s] %s", v.Severity, v.Message)))
		}
		failed = append(failed, b.String())
	}

	return strings.Join(append(passed, failed...), "\n")
}

// formatReport renders violations as text grouped by severity, or
// indented JSON if asJSON. Empty: text says "No rule violations
// found.", JSON renders "[]" not "null".
func formatReport(violations []ruleset.Violation, asJSON bool) (string, error) {
	if asJSON {
		if violations == nil {
			violations = []ruleset.Violation{}
		}
		data, err := json.MarshalIndent(violations, "", "  ")
		if err != nil {
			return "", fmt.Errorf("format report: %w", err)
		}
		return string(data), nil
	}

	if len(violations) == 0 {
		return "No rule violations found.", nil
	}

	var b strings.Builder
	for _, sev := range severityOrder {
		var group []ruleset.Violation
		for _, v := range violations {
			if v.Severity == sev {
				group = append(group, v)
			}
		}
		if len(group) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s (%d)\n", strings.ToUpper(string(sev)), len(group))
		for _, v := range group {
			fmt.Fprintf(&b, "- [%s] %s\n", v.Rule, v.Message)
		}
	}

	return strings.TrimRight(b.String(), "\n"), nil
}
