/* Package main implements the coordinator, a simple binary for executing each stage as named in the input file.
 */
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// Hardcoded module names and paths.
// For this to be actually modular, these should be fed in via config or env, ideally with enumerations to prevent executing arbitrary shell commands.
const (
	appName string = "Omen"
)

var (
	log zerolog.Logger // primary output mechanism.
)

func init() {
	{ // spool up a dev logger that respects NO_COLOR
		var nc bool
		if v, found := os.LookupEnv("NO_COLOR"); found && (strings.TrimSpace(v) != "") {
			nc = true
		}

		log = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    nc,
		})
	}
}

// run is the primary driver function for the coordinator.
// It roots the filesystem, finds all required modules, and executes them in order.
func run(cmd *cobra.Command, args []string) error {
	// ensure each arg is a valid path and collect the absolute paths of each test to run
	inputPaths, err := collectJSONPaths(args)
	if err != nil {
		return err
	}
	log.Info().Strs("input paths", inputPaths).Msg("collected input file paths")

	// capture and validate module configuration
	var modules modules
	{
		modulesCfgPath, err := cmd.Flags().GetString("modules")
		if err != nil {
			return fmt.Errorf("failed to fetch module switch: %w", err)
		}
		f, err := os.Open(modulesCfgPath)
		if err != nil {
			return err
		}
		if m, errs := ReadModuleConfig(f); len(errs) != 0 {
			// compose the errors into a clean list:
			var sb strings.Builder
			sb.WriteString("failed to generate module configuration from " + modulesCfgPath + ":\n")
			for i, err := range errs {
				fmt.Fprintf(&sb, "[%d] %s\n", i, err)
			}
			// chomp the newline
			return errors.New(sb.String()[:sb.Len()-1])
		} else {
			modules = m
		}
	}
	log.Debug().Any("modules", modules).Msg("constructed module set")

	// execute input validation against the argument
	// TODO

	return nil
}

// Gathers the .json files relevant to each path.
// For paths that point to a file, adds the file path to the list.
// For paths that point to a directory, shallowly walks the directory, adding all .json files to the list.
//
// Returns a list of absolute paths to input files.
func collectJSONPaths(argPaths []string) ([]string, error) {
	var inputPaths []string
	for i, arg := range argPaths {
		fi, err := os.Stat(arg)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %w", i, err)
		}
		if fi.IsDir() {
			// shallow walk for .json
			if err := filepath.WalkDir(arg, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if filepath.Ext(d.Name()) == ".json" {
					inputPaths = append(inputPaths, path)
				}
				return nil
			}); err != nil {
				return nil, err
			}
		} else {
			inputPaths = append(inputPaths, arg)
		}
	}
	return inputPaths, nil
}

func main() {
	// generate the command tree
	root := &cobra.Command{
		Use:   appName + " <>.json...",
		Short: appName + " is a pipeline for executing network simulation tests",
		Long: appName + ` is a helper pipeline capable of building topologies and testing them automatically.
To start a run, simply invoke this binary and give it the path to a configuration file.
Each bare argument is treated as a separate input file and thus separate run.
If a directory is given as an argument, ` + appName + ` will run all json files at the top level; it will NOT recur into subdirectories to look for json files.

Because Omen is a set of disparate module run in sequence, this binary (the Coordinator) just serves to invoke each module and ensure its input/output are prepared.

When a run starts, it is assigned a random identifier.
While modules operate independently and thus do not about correlating IDs, they can be useful for examining intermediary data structures or continuing a run if it was interrupted.`,
		RunE: run,
	}
	root.Example = appName + " topology1.json " + " topologies/"
	root.Args = cobra.MinimumNArgs(1)
	root.Flags().StringP("modules", "m", "modules.json", "path to modules.json file (the modules coordinator should launch)")

	// NOTE(rlandau): because of how cobra works, the actual main function is a stub. run() is the real "main" function
	if err := fang.Execute(context.Background(), root, fang.WithoutCompletions(), fang.WithErrorHandler(
		func(w io.Writer, styles fang.Styles, err error) {
			// we use a custom error handler as the default one transforms to title case (which collapses newlines and we don't want that)

			fmt.Fprintln(w, styles.ErrorHeader.String())
			fmt.Fprintln(w, styles.ErrorText.UnsetTransform().Render(err.Error())) //styles.ErrorText.Render(err.Error()+"."))
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

		})); err != nil {
		// fang logs returned errors for us
		os.Exit(1)
	}
}

// Borrowed from fang.go's DefaultErrorHandling.
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
