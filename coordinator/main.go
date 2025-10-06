/*
Package main implements the coordinator, a simple binary for sequentially executing each module in the Omen pipeline.

Uses hardcoded paths and commands for module execution.
*/
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	appName                string = "Omen"
	validatedDir           string = "validated_input" // intermediary directory hosting files that have been run through the validator
	inputValidatorImage    string = "omen-input-validator"
	inputValidatorImageTag string = "latest"
)

var (
	log  zerolog.Logger // primary output mechanism
	dCLI *client.Client // our docker client
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
	// check flags
	if ll, err := cmd.Flags().GetString("log-level"); err != nil {
		panic(err)
	} else if l, err := zerolog.ParseLevel(strings.ToLower(ll)); err != nil {
		return err
	} else {
		log = log.Level(l)
	}

	// ensure each arg is a valid path and collect the absolute paths of each test to run
	inputPaths, err := collectJSONPaths(args)
	if err != nil {
		return err
	} else if len(inputPaths) == 0 {
		return errors.New("no .json file where found in the given paths")
	}
	log.Info().Strs("files", inputPaths).Msg("collected input file paths")

	// run each validated file through validation
	_, err = runInputValidationModule(inputPaths)
	if err != nil {
		return err
	}
	// ensure at least 1 file made it through validation
	{
		found := false
		if err := filepath.WalkDir(validatedDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(d.Name()) == ".json" && !d.IsDir() {
				found = true
			}
			return nil
		}); err != nil {
			return err
		} else if !found {
			return ErrNoFilesValidated
		}
	}

	// execute the transport code
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

type invalidInput struct {
	Ok     bool `json:"ok"`
	Errors []struct {
		Loc  string `json:"loc"`
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"errors"`
	Warnings []struct {
		Loc  string `json:"loc"`
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"warnings"`
}

// Executes the input validator against each input path.
// Files that pass are moved to validatedDir/ and have their token prefixed.
func runInputValidationModule(inputPaths []string) (string, error) {
	tDir := path.Join(os.TempDir(), validatedDir)
	// destroy the directory
	if err := os.RemoveAll(tDir); err != nil {
		return "", err
	}
	if err := os.Mkdir(tDir, 0755); err != nil {
		// if the directory already exists, no problem, just empty it out
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	// create a directory to place validated files
	if err := os.Mkdir(validatedDir, 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", err
	}
	log.Debug().Str("path", tDir).Msg("created directory for validated inputs")
	var (
		wg     sync.WaitGroup
		tokens sync.Map // token -> input path
	)
	for i := range inputPaths {
		// as each file completes, write it into the temp directory
		wg.Go(validateIn(inputPaths[i], &tokens))
	}
	wg.Wait()

	return tDir, nil
}

// helper function intended to be called in a separate goroutine.
// Executes the input validator docker image against the given input file.
// Generates a unique token for this run, storing that token in the sync.Map.
// Validated files are placed into validatedDir.
func validateIn(iPath string, tokens *sync.Map) func() {
	return func() {
		iFile := path.Base(iPath)
		if !path.IsAbs(iPath) { // Docker requires paths to be prefixed with ./ or be absolute
			// path.Join will not prefix a ./, but we need one so goodbye Windows compatibility
			iPath = "./" + iPath
		}
		// assign a token to this file
		var token uint64
		for { // claim an unused token
			token = rand.Uint64()
			if _, loaded := tokens.LoadOrStore(token, iPath); !loaded {
				break
			}
		}
		log.Info().Str("filename", iFile).Uint64("token", token).Msgf("assigned token '%v' to input file %v", token, iPath)

		// execute input validation
		cmd := exec.Command("docker", "run", "--rm", "-v", iPath+":/input/"+path.Base(iFile), inputValidatorImage+":"+inputValidatorImageTag, "/input/"+iFile)

		log.Debug().Strs("args", cmd.Args).Msg("executing validator script")

		if stdout, err := cmd.Output(); err != nil {
			ee, ok := err.(*exec.ExitError)
			if ok && ee.ExitCode() == 1 { // the script ran successfully but the file isn't valid
				// unmarshal the data so we can present it well
				inv := invalidInput{}
				if err := json.Unmarshal(stdout, &inv); err != nil {
					log.Error().Err(err).Msg("failed to unmarshal script output as json")
					return
				}
				out := strings.Builder{}
				fmt.Fprintf(&out, "File %v has issues:\n", iPath)
				if len(inv.Errors) > 0 {
					fmt.Fprintf(&out, "%v\n", errorHeaderSty.Render("ERRORS"))
					for _, e := range inv.Errors {
						fmt.Fprintf(&out, "---%s: %s\n", e.Loc, e.Msg)
					}
				}
				if len(inv.Warnings) > 0 {
					fmt.Fprintf(&out, "%v\n", warningHeaderSty.Render("WARNINGS"))
					for _, w := range inv.Warnings {
						fmt.Fprintf(&out, "---%s: %s\n", w.Loc, w.Msg)
					}
				}

				fmt.Println(out.String())
			} else {
				log.Error().Str("stdout", string(stdout)).Err(err).Msg("failed to run input validation module")
			}
			return
		}
		// file is valid
		vPath := path.Join(validatedDir, strconv.FormatUint(token, 10)+"_"+iFile)
		log.Debug().Str("original path", iPath).Str("destination", vPath).Msg("copying file to validated directory")
		// copy the validated file into our validated directory and attack a token to it for identification
		rd, err := os.Open(iPath)
		if err != nil {
			log.Warn().Err(err).Str("original path", iPath).Msg("failed to read file")
			return
		}
		wr, err := os.Create(vPath)
		if err != nil {
			log.Warn().Err(err).Str("original path", iPath).Str("write path", vPath).Msg("failed to write validated file")
			return
		}
		if _, err := io.Copy(wr, rd); err != nil {
			log.Warn().Err(err).Str("original path", iPath).Str("write path", vPath).Msg("failed to write validated file")
			return
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
			var err error
			dCLI, err = client.NewClientWithOpts(client.FromEnv)
			if err != nil {
				return err
			}
			// TODO make this robust enough to check for existing (correctly configured) containers we can reuse.
			// spin up the containers required for visualization

			// Grafana
			if cr, err := dCLI.ContainerCreate(context.TODO(),
				&container.Config{
					ExposedPorts: nat.PortSet{nat.Port("3000/tcp"): struct{}{}},
					Image:        "grafana/grafana",
				},
				&container.HostConfig{
					PortBindings: nat.PortMap{nat.Port("3000/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "3000"}}},
				},
				nil,
				nil,
				"OmenVizGrafana",
			); err != nil {
				return fmt.Errorf("failed to start grafana container: %w", err)
			} else if len(cr.Warnings) > 0 {
				log.Warn().Strs("warnings", cr.Warnings).Str("container ID", cr.ID).Msg("spun up grafana container with warnings")
			} else {
				log.Info().Str("container ID", cr.ID).Msg("spun up grafana container")
			}
			// MySQL
			if cr, err := dCLI.ContainerCreate(context.TODO(),
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
			); err != nil {
				return fmt.Errorf("failed to start mysql container: %w", err)
			} else if len(cr.Warnings) > 0 {
				log.Warn().Strs("warnings", cr.Warnings).Str("container ID", cr.ID).Msg("spun up mysql container with warnings")
			} else {
				log.Info().Str("container ID", cr.ID).Msg("spun up mysql container")
			}

			return nil
		},
		RunE: run,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			// spit out instructions on shutting down the visualization containers
			// TODO

			return dCLI.Close()
		},
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
