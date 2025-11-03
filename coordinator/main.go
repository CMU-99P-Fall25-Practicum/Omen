/*
Package main implements the coordinator, a simple binary for sequentially executing each module in the Omen pipeline.

Uses hardcoded paths and commands for module execution.
*/
package main

import (
	omen "Omen"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Hardcoded module names and paths.
// For this to be actually modular, these should be fed in via config or env, ideally with enumerations to prevent executing arbitrary shell commands.
const (
	appName                         string = "Omen"
	inputValidatorImage             string = "0_omen-input-validator"
	inputValidatorImageTag          string = "latest"
	DefaultTestRunnerBinaryPath     string = "./1_spawn"
	DefaultCoalesceOutputBinaryPath string = "./2_output_processing"
	DefaultLoaderScriptPath         string = "omenloader.py"
)

var (
	// global logger
	log  zerolog.Logger
	dCLI *client.Client // our docker client
	// ID of the Grafana container when it is spinning so we can shut it down if the pipeline fails
	grafanaContainerID string
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
	{ // connect to the local docker engine
		var err error
		if dCLI, err = client.NewClientWithOpts(client.FromEnv); err != nil {
			log.Fatal().Err(err).Msg("failed to contact docker engine. Is docker installed and in your PATH?")
		}
	}
}

func main() {
	// define flags
	fs := pflag.FlagSet{}
	fs.String("log-level", "INFO", "set verbosity of the logger. Must be one of {TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC}.")
	fs.Uint16("grafana-port", 3000, "set the port the Grafana container should bind to")
	fs.StringP("test-runner", "1", DefaultTestRunnerBinaryPath, "override the path to the test runner binary")
	fs.StringP("coalesce-output", "2", DefaultCoalesceOutputBinaryPath, "override the path to the coalesce output binary")

	// generate the command tree
	root := &cobra.Command{
		Use:   appName + " <input>.json",
		Short: appName + " is a pipeline for executing network simulation tests",
		Long: appName + ` is a helper pipeline capable of building topologies and testing them automatically.
Because Omen is a set of disparate modules run in sequence, this binary (the Coordinator) just serves to invoke each module and ensure its input/output are prepared.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// set log level
			ll, err := fs.GetString("log-level")
			if err != nil {
				return err
			}
			l, err := zerolog.ParseLevel(ll)
			if err != nil {
				return err
			}
			log = log.Level(l)
			return nil
		},
		RunE:    run,
		Example: appName + " topology1.json ",
		Args:    cobra.ExactArgs(1), // for the time being, allow only a single file
	}
	// attach flags
	root.Flags().AddFlagSet(&fs)

	// NOTE(rlandau): because of how cobra works, the actual main function is a stub. run() is the real "main" function
	if err := fang.Execute(context.Background(), root,
		fang.WithoutCompletions(),
		fang.WithVersion(omen.Version),
		fang.WithErrorHandler(omen.FangErrorHandler)); err != nil {
		// fang logs returned errors for us
		os.Exit(1)
	}
}

// cleanup shutters the docker containers it spun up if the pipeline failed.
// Otherwise, leaves a message about still-spinning containers.
func cleanup(errored bool) {
	defer dCLI.Close()
	if errored { // force-shutter the grafana container
		if err := dCLI.ContainerRemove(context.Background(), grafanaContainerID, container.RemoveOptions{Force: true}); err != nil {
			log.Error().Err(err).Msg("failed to force-remove the Grafana container")
		}
	} else { // notify about still running containers
		fmt.Println("Remember to stop the Grafana container when you are done! ID: " + grafanaContainerID)
	}
}
