package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/pflag"
)

// default values
const (
	defaultOutputDir string = "./results"
)

// flag values
var (
	outputDir *string
)

func init() {
	outputDir = pflag.StringP("output", "o", defaultOutputDir, "directory to write processed files to")
}

func main() {
	// validate arguments
	if len(pflag.Args()) != 1 {
		fmt.Printf("Usage: %s <path_to_mn_result_raw_directory>\n", os.Args[0])
		fmt.Printf("Example: %s ../1_spawn_topology/mn_result_raw\n", os.Args[0])
		os.Exit(1)
	}
	inputDir := pflag.Arg(0)

	// Find the latest subdirectory
	latestDir, err := findLatestDirectory(inputDir)
	if err != nil {
		fmt.Printf("Error finding latest directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing files in: %s\n", latestDir)

	// prepare output dir
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating results directory: %v\n", err)
		os.Exit(1)
	}

	// collect raw output into local struct arrays
	movements, pings, stations, aps, err := processRawFileDirectory(latestDir)
	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	}

	// write the collection of all ping data
	if len(pings) > 0 {
		outputPath := filepath.Join(*outputDir, "pingall_full_data.csv")
		if err := writePingAllFull(outputPath, pings); err != nil {
			fmt.Printf("Error writing pingall CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d movements and %d ping records\n", len(movements), len(pings))
		fmt.Printf("Pingall results written to: %s\n", outputPath)
	}

	// Write iw CSV if we have station/AP data
	if len(stations) > 0 || len(aps) > 0 {
		iwOutputPath := filepath.Join(*outputDir, "final_iw_data.csv")
		if err := writeIwToCSV(iwOutputPath, stations, aps); err != nil {
			fmt.Printf("Error writing iw CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d stations and %d access points\n", len(stations), len(aps))
		fmt.Printf("IW results written to: %s\n", iwOutputPath)

		// Process nodes output (per test file)
		fmt.Println("\nGenerating per-test-file nodes CSV files:")
		_, err := processNodesOutput(stations, aps, pings, movements, *outputDir)
		if err != nil {
			fmt.Printf("Error processing nodes output: %v\n", err)
			os.Exit(1)
		}

		// Process edges output (per test file)
		fmt.Println("\nGenerating per-test-file edges CSV files:")
		if err := processEdgesOutput(pings, *outputDir); err != nil {
			fmt.Printf("Error processing edges output: %v\n", err)
			os.Exit(1)
		}
	}
}

func findLatestDirectory(basePath string) (string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %v", basePath, err)
	}

	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}

	if len(directories) == 0 {
		return "", fmt.Errorf("no subdirectories found in %s", basePath)
	}

	// Sort directories by name (assuming timestamp format makes them sortable)
	sort.Strings(directories)
	latest := directories[len(directories)-1]

	return filepath.Join(basePath, latest), nil
}
