package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/pflag"
)

// expected timestamp format in directory name
const directoryNameFormat string = "20060102_150405"

const (
	fullPingDataCSV string = "ping_data.csv" // name of the cumulative ping data file
	fullIWDataCSV   string = "final_iw_data.csv"
)

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

	// prepare output dir
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing files in: %s\n", latestDir)

	// Process all .txt files
	parsed, err := processRawFileDirectory(latestDir)
	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	} else if len(parsed) == 0 {
		fmt.Printf("no raw files were parsed\n")
		return
	}

	{ // write complete ping data from all parsed models
		op := filepath.Join(*outputDir, fullPingDataCSV)
		count, err := writePingAllFull(op, parsed)
		if err != nil {
			fmt.Printf("Error writing pingall CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d ping records\n"+
			"Pingall results written to: %s\n", count, op)
	}
	{ // write complete IW data from all parsed models
		op := filepath.Join(*outputDir, fullIWDataCSV)
		staCount, apCount, err := writeIWFull(op, parsed)
		if err != nil {
			fmt.Printf("Error writing iw CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully processed %d stations and %d access points\n", staCount, apCount)
		fmt.Printf("IW results written to: %s\n", op)
	}
	// write a folder for each timeframe
	for tf := range parsed {
		// create subdir for this timeframe
		tfDir := path.Join(*outputDir, "timeframe"+strconv.FormatUint(uint64(tf), 10))
		if err := os.Mkdir(tfDir, 0755); err != nil && !errors.Is(err, fs.ErrExist) {
			fmt.Printf("failed to create directory %s: %v\n", tfDir, err)
			os.Exit(1)
		}

		fmt.Printf("writing data from timeframe %d\n", tf)
		// process nodes for this timeframe
		err := writeNodesCSV(parsed[tf], tfDir)
		if err != nil {
			fmt.Printf("Error processing nodes output: %v\n", err)
			os.Exit(1)
		}

		// process edges for this timeframe
		if err := writeEdgesCSV(parsed[tf], tfDir); err != nil {
			fmt.Printf("Error processing edges output: %v\n", err)
			os.Exit(1)
		}
		// write position files into each timeframe
		pth := path.Join(tfDir, "ping_data_movement_"+strconv.FormatInt(int64(tf), 10)+".csv")
		if err := writeMovementCSV(pth, uint64(tf), parsed[tf]); err != nil {
			fmt.Printf("failed to write ping_data_movement file for timeframe %d: %v\n", tf, err)
			os.Exit(1)
		}
		fmt.Printf("\tPing CSV for timeframe %d written to: %s\n", tf, pth)

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
