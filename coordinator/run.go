package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

// This file contains Coordinator's driver function and its helper functions.

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

	// spin up Grafana container for visualizer
	{
		cr, err := dCLI.ContainerCreate(context.TODO(),
			&container.Config{
				ExposedPorts: nat.PortSet{nat.Port("3000/tcp"): struct{}{}},
				Image:        "grafana/grafana",
			},
			&container.HostConfig{
				PortBindings: nat.PortMap{
					nat.Port("3000/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: grafanaPortStr}},
				},
			},
			nil,
			nil,
			"OmenVizGrafana_p"+grafanaPortStr)
		if err != nil {
			return fmt.Errorf("failed to create grafana container: %w", err)
		}
		if len(cr.Warnings) > 0 {
			log.Warn().Strs("warnings", cr.Warnings).Str("container ID", cr.ID).Msg("spun up grafana container with warnings")
		} else {
			log.Info().Str("container ID", cr.ID).Msg("spun up grafana container")
		}

		if err := dCLI.ContainerStart(context.TODO(), cr.ID, container.StartOptions{}); err != nil {
			return fmt.Errorf("failed to start grafana container: %w", err)
		}
	}

	err := executePipeline(inputPath, testRunnerBinaryPath, coalesceOutputBinaryPath)
	if err == nil {
		fmt.Println("Results are available @ localhost:" + grafanaPortStr)
	}
	cleanup(err != nil)
	return err
}

func executePipeline(inputPath, testRunnerBinaryPath, coalesceOutputBinaryPath string) error {
	paths, err := runInputValidationModule([]string{inputPath})
	if err != nil {
		return err
	}

	// NOTE(rlandau): as we only accept a single file atn, `paths` should be at most 1 element

	var erred bool // an error occurred at some point
	for _, path := range paths {
		var sbErr strings.Builder

		// execute the test runner module
		log.Info().Str("path", path).Msg("executing topology tests")
		cmd := exec.Command(testRunnerBinaryPath, path)
		log.Debug().Str("path", cmd.Path).Strs("args", cmd.Args).Msg("executing test runner binary")
		cmd.Stderr = &sbErr
		if _, err := cmd.Output(); err != nil {
			log.Error().Err(err).Str("path", cmd.Path).Str("stderr", sbErr.String()).Msg("failed to run test runner binary")
			erred = true
			continue
		}

		sbErr.Reset()

		// execute coalesce output module
		log.Info().Str("path", path).Msg("coalescing raw test output")
		cmd = exec.Command(coalesceOutputBinaryPath, "mn_result_raw/")
		log.Debug().Str("path", cmd.Path).Strs("args", cmd.Args).Msg("executing coalesce output binary")
		cmd.Stderr = &sbErr
		if out, err := cmd.Output(); err != nil {
			log.Error().Err(err).Str("path", cmd.Path).Str("stdout", string(out)).Str("stderr", sbErr.String()).Msg("failed to run coalesce output binary")
			erred = true
			continue
		}
	}
	if erred {
		return errors.New("an error occurred")
	}

	var sbErr strings.Builder
	// load visualization
	vizLoaderCmd := exec.Command("docker", "run", "--rm", "-v", "./result/nodes.csv:/input/nodes.csv", "-v", "./result/edges.csv:/input/edges.csv", "3_omen-output-visualizer", "/input/nodes.csv", "/input/edges.csv")
	log.Debug().Strs("args", vizLoaderCmd.Args).Msg("executing test runner binary")
	vizLoaderCmd.Stderr = &sbErr
	if _, err := vizLoaderCmd.Output(); err != nil {
		log.Error().Err(err).Str("path", vizLoaderCmd.Path).Str("stderr", sbErr.String()).Msg("failed to run test runner binary")
		return err
	}
	// -it -e DB_HOST=172.17.0.3 -e DB_PASS=mypass -v ./result/nodes.csv:/input/nodes.csv -v ./result/edges.csv:/input/edges.csv 3_omen-output-visualizer /input/nodes.csv /input/edges.csv

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
