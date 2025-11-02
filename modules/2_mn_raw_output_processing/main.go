package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
)

// expected timestamp format in directory name
const directoryNameFormat string = "20060102_150405"

// flag values
var (
	outputDir *string
)

// init defines and maps flags
func init() {
	outputDir = pflag.StringP("output", "o", "./results", "directory to write processed files to")
}

func main() {
	pflag.Parse()
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
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Process all .txt files
	parsed, err := processRawFileDirectory(latestDir)
	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	} else if len(parsed) == 0 {
		fmt.Printf("no raw files were parsed\n")
		return
	}

	// write complete ping data from all parsed models
	{
		op := filepath.Join(*outputDir, "pingall_full_data.csv")
		if err := writePingAllFull(op, movements, pings); err != nil {
			fmt.Printf("Error writing pingall CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d movements and %d ping records\n", len(movements), len(pings))
		fmt.Printf("Pingall results written to: %s\n", op)
	}
	// generate complete IW data and individual timeframe file pairs
	if len(stations) > 0 || len(aps) > 0 {
		op := filepath.Join(*outputDir, "final_iw_data.csv")
		if err := writeIWFull(op, stations, aps); err != nil {
			fmt.Printf("Error writing iw CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d stations and %d access points\n", len(stations), len(aps))
		fmt.Printf("IW results written to: %s\n", op)

		// Process nodes output (per test file)
		fmt.Println("\nGenerating per-timeframe nodes CSV files:")
		_, err := processNodesOutput(stations, aps, pings, movements, *outputDir)
		if err != nil {
			fmt.Printf("Error processing nodes output: %v\n", err)
			os.Exit(1)
		}

		// Process edges output (per test file)
		fmt.Println("\nGenerating per-timeframe edges CSV files:")
		if err := processEdgesOutput(pings, *outputDir); err != nil {
			fmt.Printf("Error processing edges output: %v\n", err)
			os.Exit(1)
		}
		// write position files into each timeframe
		writeMovementCSV(path.Join(*outputDir, "ping_data_movement_1.csv"))

	}
}

// findLatestDirectory
func findLatestDirectory(basePath string) (string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %v", basePath, err)
	} else if len(entries) <= 0 {
		return "", fmt.Errorf("no subdirectories found in %s", basePath)
	}

	var (
		newestTime time.Time
		newestDir  string
	)
	for _, entry := range entries {
		if entry.IsDir() {
			v, err := time.Parse(directoryNameFormat, entry.Name())
			if err != nil { // if an error occurs, skip it
				continue
			} else if newestTime.Before(v) {
				newestTime = v
				newestDir = entry.Name()
			}
		}
	}
	if newestDir == "" {
		return "", fmt.Errorf("no subdirectories with the correct format found in %s", basePath)
	}

	return path.Join(basePath, newestDir), nil
}
