package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <path_to_mn_result_raw_directory>\n", os.Args[0])
		fmt.Printf("Example: %s ../1_spawn_topology/mn_result_raw\n", os.Args[0])
		os.Exit(1)
	}

	inputDir := os.Args[1]

	// Find the latest subdirectory
	latestDir, err := findLatestDirectory(inputDir)
	if err != nil {
		fmt.Printf("Error finding latest directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing files in: %s\n", latestDir)

	// Create results directory if it doesn't exist
	resultsDir := "./result"
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		fmt.Printf("Error creating results directory: %v\n", err)
		os.Exit(1)
	}

	// Process all .txt files
	movements, pings, stations, aps, err := processRawFileDirectory(latestDir)
	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	}

	// Write pingall CSV if we have movement/ping data
	if len(movements) > 0 || len(pings) > 0 {
		outputPath := filepath.Join(resultsDir, "pingall_full_data.csv")
		if err := writeToCSV(outputPath, movements, pings); err != nil {
			fmt.Printf("Error writing pingall CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d movements and %d ping records\n", len(movements), len(pings))
		fmt.Printf("Pingall results written to: %s\n", outputPath)
	}

	// Write iw CSV if we have station/AP data
	if len(stations) > 0 || len(aps) > 0 {
		iwOutputPath := filepath.Join(resultsDir, "final_iw_data.csv")
		if err := writeIwToCSV(iwOutputPath, stations, aps); err != nil {
			fmt.Printf("Error writing iw CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d stations and %d access points\n", len(stations), len(aps))
		fmt.Printf("IW results written to: %s\n", iwOutputPath)

		// Process nodes output (per test file)
		fmt.Println("\nGenerating per-test-file nodes CSV files:")
		_, err := processNodesOutput(stations, aps, pings, movements, resultsDir)
		if err != nil {
			fmt.Printf("Error processing nodes output: %v\n", err)
			os.Exit(1)
		}

		// Process edges output (per test file)
		fmt.Println("\nGenerating per-test-file edges CSV files:")
		if err := processEdgesOutput(pings, resultsDir); err != nil {
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
