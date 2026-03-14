// Package cliui provides styled terminal output for the pressluft CLI.
//
// All output functions use lipgloss.Println for automatic TTY detection
// and color profile downsampling. When stdout is not a terminal (piped,
// CI, etc.), output degrades to plain text with no ANSI codes.
//
// This package has no runtime state. All functions are pure and safe
// for concurrent use.
package cliui

import (
	"fmt"
	"os"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// Styles — pre-built lipgloss styles used across the CLI.
var (
	Bold    = lipgloss.NewStyle().Bold(true)
	Dim     = lipgloss.NewStyle().Faint(true)
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	Error   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	Accent  = lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // blue
	Label   = lipgloss.NewStyle().Faint(true).Width(22)           // right-padded key labels
)

// Header prints a bold section header.
//
//	pressluft doctor
func Header(name string) {
	lipgloss.Println(Bold.Render("pressluft " + name))
}

// KeyValue prints a dim label and its value, indented.
//
//	execution_mode  dev
func KeyValue(key, value string) {
	lipgloss.Println("  " + Label.Render(key) + value)
}

// Step prints an in-progress step indicator.
//
//	→ Generating contracts...
func Step(name string) {
	lipgloss.Println("  " + Accent.Render("→") + " " + name + "...")
}

// StepDone prints a completed step indicator.
//
//	✓ Generating contracts
func StepDone(name string) {
	lipgloss.Println("  " + Success.Render("✓") + " " + name)
}

// CheckOK formats a passing health check line.
//
//	✓ db: database opens and responds to ping
func CheckOK(name, detail string) string {
	return "  " + Success.Render("✓") + " " + Dim.Render(name+":") + " " + detail
}

// CheckWarn formats a warning health check line.
//
//	⚠ ca_key: will be created on startup
func CheckWarn(name, detail string) string {
	return "  " + Warning.Render("⚠") + " " + Dim.Render(name+":") + " " + detail
}

// CheckFail formats a failing health check line.
//
//	✗ db: open db: file not found
func CheckFail(name, detail string) string {
	return "  " + Error.Render("✗") + " " + Dim.Render(name+":") + " " + detail
}

// WarnBox prints a warning message inside a visible box.
func WarnBox(lines []string) {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 1).
		Foreground(lipgloss.Color("3"))

	lipgloss.Println(border.Render(strings.Join(lines, "\n")))
}

// Issues prints a list of issue strings, indented and prefixed.
func Issues(issues []string) {
	for _, issue := range issues {
		lipgloss.Println("  " + Error.Render("•") + " " + issue)
	}
}

// Hint prints a dim suggestion line.
//
//	To reset local state: rm -rf .pressluft
func Hint(text string) {
	lipgloss.Println(Dim.Render("  " + text))
}

// Errf prints a styled error line to stderr.
func Errf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	lipgloss.Fprintln(os.Stderr, Error.Render("error: "+msg))
}
