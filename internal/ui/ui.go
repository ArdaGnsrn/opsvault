package ui

import "github.com/fatih/color"

var (
	Green  = color.New(color.FgGreen, color.Bold)
	Red    = color.New(color.FgRed, color.Bold)
	Yellow = color.New(color.FgYellow, color.Bold)
	Cyan   = color.New(color.FgCyan, color.Bold)
	Bold   = color.New(color.Bold)
	Dim    = color.New(color.Faint)
	White  = color.New(color.FgWhite)
)

func OK(msg string) string   { return Green.Sprint("  ✓  ") + msg }
func Fail(msg string) string { return Red.Sprint("  ✗  ") + msg }
func Warn(msg string) string { return Yellow.Sprint("  ⚠  ") + msg }
func Info(msg string) string { return Cyan.Sprint("  →  ") + msg }
