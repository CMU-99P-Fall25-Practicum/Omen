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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// #region input validation

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

//#endregion input validation
