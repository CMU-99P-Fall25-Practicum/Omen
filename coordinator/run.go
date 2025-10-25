package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// This file contains Coordinator's driver function and its helper functions.

// ErrNoFilesValidated returns an error as it says on the tin
var ErrNoFilesValidated = errors.New("no files passed validation")

// runWrapped allows us to wrap the run command in an error check to ensure the proper clean up is executed
func runWrapped(cmd *cobra.Command, args []string) error {
	err := run(cmd, args)
	cleanup(err != nil)
	return err
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

	// run each path through validation
	paths, err := runInputValidationModule(inputPaths)
	if err != nil {
		return err
	}
	var erred bool // an error occurred at some point
	for _, path := range paths {
		var sbErr strings.Builder

		// execute the test runner module
		log.Info().Str("path", path).Msg("executing topology tests")
		cmd := exec.Command("./"+_1TestRunnerModuleBinary, path)
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
		cmd = exec.Command("./"+_2CoalesceOutputBinary, "mn_result_raw/")
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

	fmt.Printf("Results are available @ ???")

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
