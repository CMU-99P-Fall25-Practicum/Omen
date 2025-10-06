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
