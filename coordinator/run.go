package main

import (
	omen "Omen"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

const (
	testRunnerStdoutLog     string = "test_runner.out.log"
	testRunnerStderrLog     string = "test_runner.err.log"
	coalesceOutputStdoutLog string = "coalesce_output.out.log"
	coalesceOutputStderrLog string = "coalesce_output.err.log"
)

// ErrNoFilesValidated returns an error as it says on the tin
var ErrNoFilesValidated = errors.New("no files passed validation")

// run is the primary driver function.
// It is responsible for preparing all information, driving the pipeline, and managing docker containers.
func run(cmd *cobra.Command, args []string) error {
	var (
		grafanaPortStr           string
		testRunnerBinaryPath     string
		coalesceOutputBinaryPath string
	)
	// consume flags
	{
		grafanaPort, err := cmd.Flags().GetUint16("grafana-port")
		if err != nil {
			return err
		}
		grafanaPortStr = strconv.FormatUint(uint64(grafanaPort), 10)

		if testRunnerBinaryPath, err = cmd.Flags().GetString("test-runner"); err != nil {
			return err
		}
		if coalesceOutputBinaryPath, err = cmd.Flags().GetString("coalesce-output"); err != nil {
			return err
		}

	}
	// validate input file
	inputPath := strings.TrimSpace(args[0])
	if inputPath == "" {
		return errors.New("input path cannot be empty")
	} else if inf, err := os.Stat(inputPath); err != nil {
		return err
	} else if inf.IsDir() {
		return fmt.Errorf("input json cannot be a directory")
	}

	err := executePipeline(inputPath, testRunnerBinaryPath, coalesceOutputBinaryPath, grafanaPortStr)
	if err == nil {
		fmt.Println("Results are available @ localhost:" + grafanaPortStr)
	}
	cleanup(err != nil)
	return err
}

func executePipeline(inputPath, testRunnerBinaryPath, coalesceOutputBinaryPath, grafanaPortStr string) error {
	paths, err := runInputValidationModule([]string{inputPath})
	if err != nil {
		return err
	}

	// NOTE(rlandau): as we only accept a single file atn, `paths` should be at most 1 element
	// Further, dies on first error
	for _, path := range paths {
		log.Info().Str("path", path).Msg("validated file")

		var sbOut, sbErr strings.Builder

		// execute the test runner module
		log.Info().Str("path", path).Msg("executing topology tests")
		cmd := exec.Command(testRunnerBinaryPath, "--interactive=false", path)
		log.Debug().Str("path", cmd.Path).Strs("args", cmd.Args).Msg("executing test runner binary")
		cmd.Stdout = &sbOut
		cmd.Stderr = &sbErr
		result := make(chan error)
		go func() {
			if err := cmd.Run(); err != nil {
				log.Error().Err(err).Str("path", cmd.Path).Msg("failed to run test runner binary")
				// write the binary's outputs to files
				if err := os.WriteFile(testRunnerStdoutLog, []byte(sbOut.String()), 0644); err != nil {
					log.Error().Err(err).Msgf("failed to write %v's stdout to %v", cmd.Path, testRunnerStdoutLog)
				}
				if err := os.WriteFile(testRunnerStderrLog, []byte(sbErr.String()), 0644); err != nil {
					log.Error().Err(err).Msgf("failed to write %v's stderr to %v", cmd.Path, testRunnerStderrLog)
				}
				result <- fmt.Errorf("failed to run test runner binary (%s): %w.\nSee '%v' and `%v` for details", cmd.Path, err, testRunnerStdoutLog, testRunnerStderrLog)
				return
			}
			log.Debug().Msg("finished processing successfully")
			result <- nil
		}()

		if err := waitDisplay(result, 5); err != nil {
			return err
		}

		sbOut.Reset()
		sbErr.Reset()

		// execute coalesce output module
		log.Info().Str("path", path).Msg("coalescing raw test output")
		cmd = exec.Command(coalesceOutputBinaryPath, "mn_result_raw/")
		log.Debug().Str("path", cmd.Path).Strs("args", cmd.Args).Msg("executing coalesce output binary")
		cmd.Stdout = &sbOut
		cmd.Stderr = &sbErr
		if err := cmd.Run(); err != nil {
			log.Error().Err(err).Str("path", cmd.Path).Msg("failed to run coalesce output binary")
			// write the binary's outputs to files
			if err := os.WriteFile(coalesceOutputStdoutLog, []byte(sbOut.String()), 0644); err != nil {
				log.Error().Err(err).Msgf("failed to write %v's stdout to %v", cmd.Path, coalesceOutputStdoutLog)
			}
			if err := os.WriteFile(coalesceOutputStderrLog, []byte(sbErr.String()), 0644); err != nil {
				log.Error().Err(err).Msgf("failed to write %v's stderr to %v", cmd.Path, coalesceOutputStderrLog)
			}
			return fmt.Errorf("failed to run coalesce output binary (%s): %w.\nSee '%v' and `%v` for details", cmd.Path, err, coalesceOutputStdoutLog, coalesceOutputStderrLog)
		}
	}

	var sbErr strings.Builder
	// generate the database
	{
		cmd := exec.Command("python3", DefaultLoaderScriptPath, "graph",
			"--db", "omen.db",
			"--recreate",
			"--root", "./results",
			"--set1-prefix", "netA", "--set1-dir", "timeframe0", "--set1-ts", "timeframe0/ping_data_movement_0.csv",
			"--set2-prefix", "netB", "--set2-dir", "timeframe1", "--set2-ts", "timeframe1/ping_data_movement_1.csv",
			"--set3-prefix", "netC", "--set3-dir", "timeframe2", "--set3-ts", "timeframe2/ping_data_movement_2.csv",
		)
		log.Debug().Strs("args", cmd.Args).Msg("executing visualization loader binary (graph)")
		cmd.Stderr = &sbErr
		if _, err := cmd.Output(); err != nil {
			log.Error().Err(err).Msg("failed to run visualization loader module (graph)")
			return errors.New(sbErr.String())
		}
	}
	sbErr.Reset()
	const dbOut string = "omen.db"
	{
		cmd := exec.Command("python3", DefaultLoaderScriptPath, "timeseries",
			"--root", "./results",
			"--csv", "ping_data.csv",
			"--db", dbOut,
			"--table", "ping_data",
			"--if-exists", "replace",
			"--aggregate-by", "movement_number",
		)
		log.Debug().Strs("args", cmd.Args).Msg("executing visualization loader binary (graph)")
		cmd.Stderr = &sbErr
		if _, err := cmd.Output(); err != nil {
			log.Error().Err(err).Msg("failed to run visualization loader module (graph)")
			return errors.New(sbErr.String())
		}
	}
	sbErr.Reset()

	// because host mounts must be absolute, we need to get the full path to the local file first
	abspth, err := filepath.Abs("omen.db")
	if err != nil {
		return err
	}

	// boot visualization container
	cr, err := dCLI.ContainerCreate(context.TODO(),
		&container.Config{
			ExposedPorts: nat.PortSet{nat.Port("3000/tcp"): struct{}{}},
			Image:        omen.VisualizationGrafanaImage,
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				nat.Port("3000/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: grafanaPortStr}},
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: abspth,
					Target: "/var/lib/grafana/data.db",
				},
			},
		},
		nil,
		nil,
		"OmenVizGrafana_p"+grafanaPortStr)
	if err != nil {
		return fmt.Errorf("failed to create grafana container: %w", err)
	}
	if len(cr.Warnings) > 0 {
		log.Warn().Strs("warnings", cr.Warnings).Str("container ID", cr.ID).Msg("created grafana container with warnings")
	} else {
		log.Info().Str("container ID", cr.ID).Msg("created grafana container")
	}

	if err := dCLI.ContainerStart(context.Background(), cr.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to spin up grafana container: %w", err)
	}
	grafanaContainerID = cr.ID

	return nil
}

// waitDisplay awaits any value on the result channel.
// In the meantime, it prints a simple, looping string to represent that processing is still occurring.
//
// charLimit sets the max number of characters to display at once.
func waitDisplay(result <-chan error, charLimit uint16) error {
	onScreen := uint16(0)
	char1, char2 := '.', ':' // the characters to alternate between
	curChar := char1
	var err error
DoneLoop:
	for {
		select {
		case err = <-result:
			// wipe away the spinner
			fmt.Printf("\r%s", strings.Repeat(" ", int(charLimit)))
			break DoneLoop
		case <-time.After(3 * time.Second):
			if onScreen > charLimit+1 { // reset and flip
				fmt.Print("\r")
				onScreen = 0
				if curChar == char1 {
					curChar = char2
				} else {
					curChar = char1
				}
			} else {
				fmt.Printf("%c", curChar)
				onScreen += 1
			}
		}
	}
	if err != nil {
		return err
	}

	return nil
}

// Gathers the .json files relevant to each path.
// For paths that point to a file, adds the file path to the list.
// For paths that point to a directory, shallowly walks the directory, adding all .json files to the list.
//
// Returns a list of absolute paths to input files.
/*func collectJSONPaths(argPaths []string) ([]string, error) {
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
}*/

// #region input validation

// InvalidInput maps to the JSON spit out after a run of input validation.
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
//
// Returns an array of paths for files that passed validation.
//
// NOTE(rlandau): assumes a unix-like host for path prefixing
func runInputValidationModule(inputPaths []string) ([]string, error) {
	var passed []string

	for _, inPath := range inputPaths {
		if strings.TrimSpace(inPath) == "" {
			continue
		}
		filename := path.Base(inPath)
		// Docker requires paths to be prefixed with ./ or be absolute
		if !path.IsAbs(inPath) && !strings.HasPrefix(inPath, "./") {
			inPath = "./" + inPath
		}
		// execute input validation
		cmd := exec.Command("docker", "run", "--rm", "-v", inPath+":/input/"+path.Base(filename), inputValidatorImage+":"+inputValidatorImageTag, "/input/"+filename)
		log.Debug().Strs("args", cmd.Args).Msg("executing validator script")
		if stdout, err := cmd.Output(); err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok || ee.ExitCode() != 1 {
				log.Error().Str("file path", inPath).Str("stdout", string(stdout)).Err(err).Msg("failed to run input validation module")
			} else { // the script ran successfully but the file isn't valid
				// unmarshal the data so we can present it well
				inv := invalidInput{}
				if err := json.Unmarshal(stdout, &inv); err != nil {
					log.Error().Err(err).Msg("failed to unmarshal script output as json")
					continue
				}
				out := strings.Builder{}
				fmt.Fprintf(&out, "File %v has issues:\n", inPath)
				if len(inv.Errors) > 0 {
					fmt.Fprintf(&out, "%v\n", omen.ErrorHeaderSty.Render("ERRORS"))
					for _, e := range inv.Errors {
						fmt.Fprintf(&out, "---%s: %s\n", e.Loc, e.Msg)
					}
				}
				if len(inv.Warnings) > 0 {
					fmt.Fprintf(&out, "%v\n", omen.WarningHeaderSty.Render("WARNINGS"))
					for _, w := range inv.Warnings {
						fmt.Fprintf(&out, "---%s: %s\n", w.Loc, w.Msg)
					}
				}

				fmt.Println(out.String())
			}
			continue
		}

		// the file is valid, add it to the list
		passed = append(passed, inPath)

	}

	if len(passed) == 0 {
		return nil, ErrNoFilesValidated
	}

	return passed, nil
}

//#endregion input validation
