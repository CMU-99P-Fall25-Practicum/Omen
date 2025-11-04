// Package omen provides functionality shared across all modules (including the magefile).
// It has no direct function within the pipeline.
package omen

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
)

const (
	Version string = "MS3"
)

// Docker-related
const (
	InputValidatorImage       string = "0_omen-input-validator"
	VisualizationLoaderImage  string = "3_omen-output-visualizer-loader"
	VisualizationGrafanaImage string = "3_omen-output-visualizer-grafana"
)

// Contains styles we are using.
// NOTE(rlandau): Most are ripped out of charmtone and based on the default color schemes Fang uses.

var (
	ErrorHeaderSty = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFAF1")).
			Background(lipgloss.Color("#FF388B")).
			Padding(0, 1).
			Bold(true)
	WarningHeaderSty = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFAF1")).
				Background(lipgloss.Color("#fff348")).
				Padding(0, 1).
				Bold(true)
)

func FangErrorHandler(w io.Writer, styles fang.Styles, err error) {
	// we use a custom error handler as the default one transforms to title case (which collapses newlines and we don't want that)
	fmt.Fprintln(w, ErrorHeaderSty.Margin(1).MarginLeft(2).Render("ERROR"))
	fmt.Fprintln(w, styles.ErrorText.UnsetTransform().Render(err.Error()))
	fmt.Fprintln(w)
	if isUsageError(err) {
		_, _ = fmt.Fprintln(w, lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.ErrorText.UnsetWidth().Render("Try"),
			styles.Program.Flag.Render(" --help "),
			styles.ErrorText.UnsetWidth().UnsetMargins().UnsetTransform().Render("for usage."),
		))
		_, _ = fmt.Fprintln(w)
	}

}

// Borrowed from fang.go's DefaultErrorHandling.
//
// XXX: this is a hack to detect usage errors.
// See: https://github.com/spf13/cobra/pull/2266
func isUsageError(err error) bool {
	s := err.Error()
	for _, prefix := range []string{
		"flag needs an argument:",
		"unknown flag:",
		"unknown shorthand flag:",
		"unknown command",
		"invalid argument",
	} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
