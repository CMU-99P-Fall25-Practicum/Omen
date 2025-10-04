//go:build mage

// Package mage implements a mage file capable of generating each module.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	buildDir string = "artefacts"

	coordinatorBin   string = "coordinator"
	spawnTopoBin     string = "1_spawn"
	outputProcessBin string = "2_output_processing"
)

var Default = Build

//#region module building

// BuildCoordinator generates the coordinator binary and sits it in ./artefacts/
func BuildCoordinator() error {
	mg.Deps(artefactDirectoryExists)
	_, err := sh.Exec(nil, nil, nil, "go", "build", "-o", "artefacts/"+coordinatorBin, "./coordinator")
	return err
}

// DockerizeIV recompiles the input validation docker container.
func DockerizeIV() error {
	mg.Deps(dockerInPath)
	return sh.Run("docker", "build", "-t", "omen-input-validator", "modules/0_input/")
}

// BuildSpawnTopo builds the binary for the glue module.
func BuildSpawnTopo() error {
	mg.Deps(artefactDirectoryExists)
	return sh.Run("go", "build", "-C", "modules/1_spawn_topology/", "-o", "../../artefacts/"+spawnTopoBin)
}

// BuildOutputProcessing builds the binary for the output coalesce module.
func BuildOutputProcessing() error {
	mg.Deps(artefactDirectoryExists)

	var sbErr strings.Builder

	_, err := sh.Exec(nil, nil, &sbErr, "go", "build", "-C", "modules/2_mn_raw_output_processing/", "-o", "../../artefacts/"+outputProcessBin)
	if err != nil {
		fmt.Println(sbErr.String())
	}
	return err
}

// DockerizeOV recompiles the output visualization docker container.
func DockerizeOV() error {
	mg.Deps(dockerInPath)
	return sh.Run("docker", "build", "-t", "3_omen-output-visualizer", "modules/3_output_visualization")
}

//#endregion module building

// Build builds all required files and containers.
func Build() {
	mg.Deps(DockerizeIV, BuildCoordinator, BuildSpawnTopo, BuildOutputProcessing, DockerizeOV)
}

// Clean deletes the build directory and everything in it.
func Clean() {
	sh.Rm(buildDir)
}

//#region helper functions

// ensures Docker is in path
func dockerInPath() error {
	_, err := exec.LookPath("docker")
	return err
}

// checks that the top-level artefact directory exists and creates it if it doesn't.
func artefactDirectoryExists() error {
	if err := os.Mkdir(buildDir, 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	return nil
}

//#endregion helper functions
