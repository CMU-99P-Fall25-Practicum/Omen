// Package mage implements a mage file capable of generating each module.
package main

import (
	omen "Omen"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
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
	var sbErr strings.Builder
	_, err := sh.Exec(nil, nil, &sbErr, "go", "build", "-o", "artefacts/"+coordinatorBin, "./coordinator")
	if err != nil {
		fmt.Fprintln(&sbErr)
	}
	return err
}

// DockerizeIV recompiles the input validation docker container.
func DockerizeIV() error {
	mg.Deps(dockerInPath)
	return sh.Run("docker", "build", "-t", omen.InputValidatorImage, "modules/0_input/")
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

// DockerizeOV recompiles the output visualization loader  and grafana-sqlite images.
func DockerizeOV() error {
	mg.Deps(dockerInPath)
	if err := sh.Run("docker", "build", "-t", omen.VisualizationLoaderImage, "-f", "modules/3_output_visualization/loader.Dockerfile", "modules/3_output_visualization"); err != nil {
		return err
	}
	return sh.Run("docker", "build", "-t", omen.VisualizationGrafanaImage, "-f", "modules/3_output_visualization/grafana-sqlite.Dockerfile", "modules/3_output_visualization")
}

//#endregion module building

// Gui leverages wails to compile the GUI.
func Gui(debug bool) error {
	if _, err := exec.LookPath("wails"); err != nil {
		return err
	}

	if err := os.Chdir("omen-gui"); err != nil {
		return err
	}
	args := []string{"build"}
	if debug {
		args = append(args, "-debug")
	}
	if err := sh.Run("wails", args...); err != nil {
		return err
	}
	if err := os.Chdir(".."); err != nil {
		return err
	}
	// copy the linux/mac binary into artefacts
	if err := sh.Copy(path.Join("artefacts", "omen-gui"), path.Join("omen-gui", "build", "bin", "omen-gui")); err != nil {
		return err
	}

	return nil
}

// Build builds all required files and containers.
func Build() error {
	mg.Deps(DockerizeIV, BuildCoordinator, BuildSpawnTopo, BuildOutputProcessing, DockerizeOV)

	// copy the driver script into the artefacts directory so it can be passed by spawn topology
	if err := sh.Copy(path.Join(buildDir, "mininet-script.py"), "modules/1_spawn_topology/mininet-script.py"); err != nil {
		return err
	}
	// copy the database generator script into artefacts for coordinator to invoke directly
	if err := sh.Copy(path.Join(buildDir, "omenloader.py"), "modules/3_output_visualization/omenloader.py"); err != nil {
		return err
	}
	return nil
}

// Clean deletes the build directory and everything in it.
func Clean() error {
	return sh.Rm(buildDir)
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
