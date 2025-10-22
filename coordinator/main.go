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
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// Hardcoded module names and paths.
// For this to be actually modular, these should be fed in via config or env, ideally with enumerations to prevent executing arbitrary shell commands.
const (
	appName                  string = "Omen"
	validatedDir             string = "validated_input" // intermediary directory hosting files that have been run through the validator
	inputValidatorImage      string = "0_omen-input-validator"
	inputValidatorImageTag   string = "latest"
	_1TestRunnerModuleBinary string = "1_spawn"
)

var (
	log  zerolog.Logger // primary output mechanism
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
	// generate the command tree
	root := &cobra.Command{
		Use:   appName + " <>.json...",
		Short: appName + " is a pipeline for executing network simulation tests",
		Long: appName + ` is a helper pipeline capable of building topologies and testing them automatically.
Each bare argument is treated as a separate input file and thus separate run.
If a directory is given as an argument, ` + appName + ` will run all json files at the top level; it will NOT recur into subdirectories to look for json files.

Because Omen is a set of disparate module run in sequence, this binary (the Coordinator) just serves to invoke each module and ensure its input/output are prepared.

When a run starts, it is assigned a random identifier.
While modules operate independently and thus do not care about correlating IDs, IDs can be useful for examining intermediary data structures or continuing a run if it was interrupted.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// TODO make this robust enough to check for existing (correctly configured) containers we can reuse.
			// spin up the containers required for visualization
			{ // Grafana
				cr, err := dCLI.ContainerCreate(context.TODO(),
					&container.Config{
						ExposedPorts: nat.PortSet{nat.Port("3000/tcp"): struct{}{}},
						Image:        "grafana/grafana",
					},
					&container.HostConfig{
						PortBindings: nat.PortMap{nat.Port("3000/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "3000"}}},
					},
					nil,
					nil,
					"OmenVizGrafana")
				if err != nil {
					return fmt.Errorf("failed to start grafana container: %w", err)
				}
				if len(cr.Warnings) > 0 {
					log.Warn().Strs("warnings", cr.Warnings).Str("container ID", cr.ID).Msg("spun up grafana container with warnings")
				} else {
					log.Info().Str("container ID", cr.ID).Msg("spun up grafana container")
				}
				containers[cr.ID] = "grafana"
				if err := dCLI.ContainerStart(context.TODO(), cr.ID, container.StartOptions{}); err != nil {
					return err
				}
			}
			{ // MySQL
				cr, err := dCLI.ContainerCreate(context.TODO(),
					&container.Config{
						ExposedPorts: nat.PortSet{nat.Port("3306/tcp"): struct{}{}},
						Env:          []string{"MYSQL_DATABASE=test", "MYSQL_ROOT_PASSWORD=mypass"},
						Image:        "mysql",
					},
					&container.HostConfig{
						PortBindings: nat.PortMap{nat.Port("33306/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "3306"}}},
					},
					nil,
					nil,
					"OmenVizSQL",
				)
				if err != nil {
					return fmt.Errorf("failed to start mysql container: %w", err)
				}
				if len(cr.Warnings) > 0 {
					log.Warn().Strs("warnings", cr.Warnings).Str("container ID", cr.ID).Msg("spun up mysql container with warnings")
				} else {
					log.Info().Str("container ID", cr.ID).Msg("spun up mysql container")
				}
				containers[cr.ID] = "sql"

				if err := dCLI.ContainerStart(context.TODO(), cr.ID, container.StartOptions{}); err != nil {
					return err
				}
			}
			return nil
		},
		RunE:     run,
		PostRunE: cleanup,
	}
	root.Example = appName + " topology1.json " + " topologies/"
	root.Args = cobra.MinimumNArgs(1)
	// establish flags
	root.Flags().String("log-level", "DEBUG", "Set verbosity of the logger. Must be one of {TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC}.")

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
// cleanup closes the connection to docker and spits out a message about what containers are still running.
func cleanup(cmd *cobra.Command, args []string) error {
	defer dCLI.Close()
	// check if our visualization containers are still running and note their IDs if they are
	runningContainers, err := dCLI.ContainerList(context.TODO(), container.ListOptions{})
	if err != nil {
		return err
	}
	// find the containers with our cached IDs
	stillRunning := []string{} // array of IDs of containers we spun up that are still spinning

	for _, cntr := range runningContainers {
		if _, found := containers[cntr.ID]; found {
			stillRunning = append(stillRunning, cntr.ID)
		}
	}
	var sb strings.Builder
	sb.WriteString("Pipeline has completed.\n")
	if len(stillRunning) > 0 {
		fmt.Fprintf(&sb, "The following %d containers were left running.\n"+
			"Remember to stop them when you are done.\n", len(stillRunning))
		for _, id := range stillRunning {
			sb.WriteString(id + " - " + containers[id] + "\n")
		}
	}
	fmt.Print(sb.String())
	return nil
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
