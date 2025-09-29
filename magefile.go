//go:build mage

// Package mage implements a mage file capable of generating each module.
package main

import (
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// DockerizeInput recompiles the input validation docker container.
func DockerizeInput() error {
	mg.Deps(dockerInPath)
	return sh.Run("docker", "build", "-t", "omen-input-validator", "modules/0_input/")
}

func dockerInPath() error {
	_, err := exec.LookPath("docker")
	return err
}
