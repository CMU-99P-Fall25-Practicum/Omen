/* Package main implements the coordinator, a simple binary for executing each stage as named in the input file.
 */
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// Hardcoded module names and paths.
// For this to be actually modular, these should be fed in via config or env, ideally with enumerations to prevent executing arbitrary shell commands.
const (
	inputModulePath = "modules/0_input/inputs.py"
)

var (
	log  zerolog.Logger
	root *cobra.Command
)

func init() {
	// spool up a dev logger
	log = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})
	// generate the command tree
	root = &cobra.Command{
		Use:   "Omen",
		Short: "Omen is a pipeline for executing network simulation tests",
		Long:  `Longform description is still TODO`,
		Run:   omen,
	}

}

// omen is the primary driver function for the coordinator.
// It roots the filesystem, finds all required modules, and executes them in order.
func omen(cmd *cobra.Command, args []string) {
	// root the fs here
	fs, err := os.OpenRoot(".")
	if err != nil {
		log.Error().Msg("failed to establish pwd as root")
	}
	// ensure we have each an executable for each stage we want to invoke
	if _, err := fs.Stat(inputModulePath); err != nil {
		log.Error().Err(err).Str("path", inputModulePath).Str("module", "input").Msg("failed to stat module")
		return
	}
	// TODO
}

func main() {
	// NOTE(rlandau): because of how cobra works, the actual main function is a stub. omen() is the real "main" function
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
}
