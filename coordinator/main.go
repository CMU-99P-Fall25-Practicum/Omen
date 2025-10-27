/*
Package main implements the coordinator, a simple binary for sequentially executing each module in the Omen pipeline.

Uses hardcoded paths and commands for module execution.
*/
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
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
)

var (
	// global logger
	log  zerolog.Logger
	dCLI *client.Client // our docker client
	// hosts information about containers we spin up and down as part of the pipeline.
	// container ID -> container name/purpose
	containers map[string]string = make(map[string]string)
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
		dCLI, err = client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to contact docker engine. Is docker installed and in your PATH?")
		}

	}
}

func main() {
	// define flags
	fs := pflag.FlagSet{}
	fs.String("log-level", "INFO", "set verbosity of the logger. Must be one of {TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC}.")
	fs.Uint16("grafana-port", 3000, "set the port the Grafana container should bind to")
	fs.StringP("test-runner", "2", DefaultTestRunnerBinaryPath, "override the path to the test runner binary")
	fs.StringP("coalesce-output", "3", DefaultCoalesceOutputBinaryPath, "override the path to the coalesce output binary")

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
		fang.WithVersion("MS2"),
		fang.WithErrorHandler(
			func(w io.Writer, styles fang.Styles, err error) {
				// we use a custom error handler as the default one transforms to title case (which collapses newlines and we don't want that)
				fmt.Fprintln(w, errorHeaderSty.Margin(1).MarginLeft(2).Render("ERROR"))
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

			})); err != nil {
		// fang logs returned errors for us
		os.Exit(1)
	}
}

// Intended to be run as a PostRunE.
// cleanup shutters the docker containers it spun up if the pipeline failed.
// Leaves a message about still-spinning containers otherwise.
func cleanup(errored bool) {
	defer dCLI.Close()
	// check if our visualization containers are still running and note their IDs if they are
	runningContainers, err := dCLI.ContainerList(context.TODO(), container.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch list of containers")
		return
	}
	// find the containers with our cached IDs
	stillRunning := []string{} // array of IDs of containers we spun up that are still spinning

	for _, cntr := range runningContainers {
		// check if any of the names contain our prefix
		for _, name := range cntr.Names {
			if strings.Contains(strings.ToLower(name), "omen") {
				stillRunning = append(stillRunning, cntr.ID)
				break
			}
		}
	}

	if errored { // shutter all containers
		for _, cntrID := range stillRunning {
			if err := dCLI.ContainerStop(context.Background(), cntrID, container.StopOptions{}); err != nil {
				log.Error().Err(err).Msgf("failed to stop container %v", cntrID)
			}

			if err := dCLI.ContainerRemove(context.TODO(), cntrID, container.RemoveOptions{}); err != nil {
				log.Error().Err(err).Msgf("failed to remove container %v", cntrID)
			}
		}
		return
	}
	// notify about still runnning containers
	var sb strings.Builder
	if len(stillRunning) > 0 {
		fmt.Fprintf(&sb, "The following %d containers were left running.\n"+
			"Remember to stop them when you are done.\n", len(stillRunning))
		for _, id := range stillRunning {
			sb.WriteString(id + " - " + containers[id] + "\n")
		}
	}
	fmt.Print(sb.String())
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
