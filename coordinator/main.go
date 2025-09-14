/* Package main implements the coordinator, a simple binary for executing each stage as named in the input file.
 */
package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// Hardcoded module names and paths.
// For this to be actually modular, these should be fed in via config or env, ideally with enumerations to prevent executing arbitrary shell commands.
const (
	inputModulePath string = "modules/0_input/inputs.py"
	appName         string = "Omen"
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
	// generate the command tree
	root := &cobra.Command{
		Use:   appName,
		Short: appName + " is a pipeline for executing network simulation tests",
		Long: appName + ` is a helper pipeline capable of building topologies and testing them automatically.
To start a run, simply invoke this binary and give it an input file.
You may control the output and execution via a limited selection of flags.
Because Omen is a set of disparate module run in sequence, this binary (the Coordinator) just serves to invoke each module and ensure its input/output are prepared.

The set of modules composing the pipeline can be tweaked by creating a modules.json file and invoking ` + appName + ` with it using the -m switch.
NOTE: the prototype does not provide alternative modules.

When a run starts, it is assigned a random identifier.
While modules operate independently and thus do not about correlating IDs, they can be useful for examining intermediary data structures or continuing a run if it was interrupted.`,
		Run: omen,
	}

	// NOTE(rlandau): because of how cobra works, the actual main function is a stub. omen() is the real "main" function
	if err := fang.Execute(context.Background(), root); err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(1)
	}
}
