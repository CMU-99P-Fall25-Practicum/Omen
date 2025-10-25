package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"strings"
)

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
// Files that pass are moved to validatedDir with their tokens prefixed.
//
// Returns an array of paths for files that passed validation
func runInputValidationModule(inputPaths []string) ([]string, error) {
	var passed []string

	for _, iPath := range inputPaths {
		iFile := path.Base(iPath)
		if !path.IsAbs(iPath) { // Docker requires paths to be prefixed with ./ or be absolute
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
					continue
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
			continue
		}

		// the file is valid, add it to the list
		passed = append(passed, iPath)

	}

	if len(passed) == 0 {
		return nil, ErrNoFilesValidated
	}

	return passed, nil
}

//#endregion input validation
