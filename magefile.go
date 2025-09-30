//go:build mage

// Package mage implements a mage file capable of generating each module.
package main

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const buildDir string = "artefacts"

var Default = Build

// DockerizeIV recompiles the input validation docker container.
func DockerizeIV() error {
	mg.Deps(dockerInPath)
	return sh.Run("docker", "build", "-t", "omen-input-validator", "modules/0_input/")
}

// ensures Docker is in path
func dockerInPath() error {
	_, err := exec.LookPath("docker")
	return err
}

// BuildCoordinator generates the coordinator binary and sits it in ./artefacts/
func BuildCoordinator() error {
	mg.Deps(artefactDirectoryExists)
	_, err := sh.Exec(nil, nil, nil, "go", "build", "-o", "artefacts/coordinator", "./coordinator")
	return err
}

// checks that the top-level artefact directory exists and creates it if it doesn't.
func artefactDirectoryExists() error {
	if err := os.Mkdir(buildDir, 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	return nil
}

// Build builds all required files and containers.
func Build() {
	mg.Deps(DockerizeIV, BuildCoordinator)
}
