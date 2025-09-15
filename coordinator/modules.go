package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ErrInvalidEnumeration returns an error string indicating badEnum is not in the allowable set for module.
func ErrInvalidEnumeration(module, badEnum string, allowable []string) error {
	return fmt.Errorf("invalid value in %s stdin ('%s'). Allowable values: %v", module, badEnum, allowable)
}

// modules represents the set of modules to run.
type modules struct {
	ZeroInput struct { // 0-input, the module responsible for validating the user input file.
		Path string `json:"path"` // path to module executable
	} `json:"0-input"`
}

// set of valid inputs for the 0-input module.
// all values are lower-cased before checking.
var zeroInputsValidInputs = []string{"user input"}

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
		// check path
		if fi, err := os.Stat(m.ZeroInput.Path); err != nil {
			errs = append(errs, fmt.Errorf("failed to stat 0-Input binary at '%s': %w", m.ZeroInput.Path, err))
		} else if fi.Mode()&0111 == 0 {
			errs = append(errs, fmt.Errorf("0-Input binary ('%s') is not executable by anyone", m.ZeroInput.Path))
		}
	}

	return
}
