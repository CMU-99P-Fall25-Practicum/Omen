package main

// Contains styles we are using.
// NOTE(rlandau): Most are ripped out of charmtone and based on the default color schemes Fang uses.

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	errorHeaderSty = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFAF1")).
			Background(lipgloss.Color("#FF388B")).
			Padding(0, 1).
			Bold(true)
	warningHeaderSty = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFAF1")).
				Background(lipgloss.Color("#fff348")).
				Padding(0, 1).
				Bold(true)
)
