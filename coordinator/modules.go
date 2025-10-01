package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
)

// modules represents the set of modules to run.
type modules struct {
	ZeroInput struct { // 0-input, the module responsible for validating the user input file.
		Path         string `json:"path"` // path to module executable
		relativePath bool   // set after stating the path
	} `json:"0-input"`
}

// ReadModuleConfig unmarshals a modules struct from the reader and validates the inputs of each module.
// If no errors are returned, caller may assume m is valid and ready for use.
//
// NOTE(rlandau): paths are tested for existence and any execute bit; this subroutine does NOT test if the process itself has execute permission.
func ReadModuleConfig(cfg io.Reader) (m modules, errs []error) {
	// slurp reader into m
	dc := json.NewDecoder(cfg)
	if err := dc.Decode(&m); err != nil {
		return m, []error{err}
	}
	// validate enumerations
	{ // 0-inputs
		// check that the path is executable (locally (with './' notation) or via $PATH)
		abs, err := exec.LookPath(m.ZeroInput.Path)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) { // if permission error, add a suggestion
				err = fmt.Errorf("%w; is the target executable?", err)
			}
			errs = append(errs, err)
		} else {
			m.ZeroInput.Path = abs
		}
	}

	return
}
