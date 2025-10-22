package main

import (
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
	"sync/atomic"

	"github.com/magefile/mage/sh"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// This file contains Coordinator's driver function and its helper functions.

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
		// coalesce output
		// TODO
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
// Files that pass are moved to validatedDir with their tokens prefixed.
//
// Returns a sync.Map of (token -> validated file) and an error.
func runInputValidationModule(inputPaths []string) (*sync.Map, error) {
	// create a directory to place validated files
	if err := os.Mkdir(validatedDir, 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}
	var (
		wg          sync.WaitGroup
		resultPaths sync.Map      // unique token -> input path (map guarantees uniqueness)
		passed      atomic.Uint64 // # of files that passed validation
	)
	for i := range inputPaths {
		// as each file completes, write it into the temp directory and add it to the list of validated files
		wg.Go(validateIn(inputPaths[i], &resultPaths, &passed))
	}
	wg.Wait()

	if passed.Load() == 0 {
		return nil, ErrNoFilesValidated
	}

	return &resultPaths, nil
}

// helper function intended to be called in a separate goroutine.
// Executes the input validator docker image against the given input file.
// Generates a unique token for this run, storing that token in the sync.Map.
// Validated files are placed into validatedDir.
func validateIn(iPath string, result *sync.Map, passed *atomic.Uint64) func() {
	return func() {
		iFile := path.Base(iPath)
		if !path.IsAbs(iPath) { // Docker requires paths to be prefixed with ./ or be absolute
			// path.Join will not prefix a ./, but we need one so goodbye Windows compatibility
			iPath = "./" + iPath
		}
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

		// assign a token to this file
		var token uint64
		for { // claim an unused token
			token = rand.Uint64()
			if _, loaded := result.LoadOrStore(token, iPath); !loaded {
				break
			}
		}
		vPath := path.Join(validatedDir, strconv.FormatUint(token, 10)+"_"+iFile)
		log.Debug().Str("original path", iPath).Str("destination", vPath).Uint64("token", token).Msg("copying file to validated directory")
		// copy the validated file into our validated directory and attach a token to it for identification
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
		// replace the path in the map with validated path
		if swapped := result.CompareAndSwap(token, iPath, vPath); !swapped {
			log.Error().Err(err).Str("input path", iPath).Str("validated path", vPath).Uint64("token", token).Msg("failed to replace input path with validated path")
		}
		passed.Add(1)
	}
}
