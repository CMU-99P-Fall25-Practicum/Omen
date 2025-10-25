package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// This file contains Coordinator's driver function and its helper functions.

// ErrNoFilesValidated returns an error as it says on the tin
var ErrNoFilesValidated = errors.New("no files passed validation")

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

	// for each validated file, execute its tests and coalesce its output
	var rangeErr error
	paths.Range(func(key, value any) bool {
		token, ok := key.(uint64)
		if !ok {
			log.Warn().Any("key", key).Msg("failed to cast key to a uint64")
			return true
		}
		validatedPath, ok := value.(string)
		if !ok {
			log.Warn().Any("value", value).Msg("failed to cast value to a string")
			return true
		}
		// execute the test runner module
		log.Info().Uint64("token", token).Str("path", validatedPath).Msg("executing topology tests")
		var _, sbErr strings.Builder
		if _, err := sh.Exec(nil, nil, &sbErr, "./"+_1TestRunnerModuleBinary, validatedPath); err != nil {
			log.Error().Str("stderr", sbErr.String()).Msg("failed to run test runner module")
			rangeErr = fmt.Errorf("failed to run test runner module '%v': %w", "./"+_1TestRunnerModuleBinary, err)
			return false
		}
		sbErr.Reset()
		// coalesce output
		if _, err := sh.Exec(nil, nil, &sbErr, "./"+_2CoalesceOutputBinary, "mn_result_raw"); err != nil {
			log.Error().Str("stderr", sbErr.String()).Msg("failed to run coalesce output module")
			rangeErr = fmt.Errorf("failed to run coalesce output module '%v':%w", "./"+_2CoalesceOutputBinary, err)
			return false
		}
		return true
	})
	if rangeErr != nil {
		return rangeErr
	}

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
