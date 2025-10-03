package main

import "errors"

// ErrNoFilesValidated returns an error as it says on the tin
var ErrNoFilesValidated = errors.New("no files passed validation")
