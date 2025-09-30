package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type MovementRecord struct {
	MovementNumber string
	NodeName       string
	Position       string
	TestFile       string
}

type PingRecord struct {
	MovementNumber string
	TestFile       string
	Src            string
	Dst            string
	Tx             string
	Rx             string
	LossPct        string
	AvgRttMs       string
}

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
	movements, pings, err := processFiles(latestDir)
	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	}

	// Write to CSV
	outputPath := filepath.Join(resultsDir, "pingall_full_data.csv")
	if err := writeToCSV(outputPath, movements, pings); err != nil {
		fmt.Printf("Error writing CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully processed %d movements and %d ping records\n", len(movements), len(pings))
	fmt.Printf("Results written to: %s\n", outputPath)
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

func processFiles(directory string) ([]MovementRecord, []PingRecord, error) {
	var movements []MovementRecord
	var pings []PingRecord

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".txt") {
			fmt.Printf("Processing file: %s\n", d.Name())

			fileMovements, filePings, err := processFile(path, d.Name())
			if err != nil {
				fmt.Printf("Warning: Error processing file %s: %v\n", d.Name(), err)
				return nil // Continue with other files
			}

			movements = append(movements, fileMovements...)
			pings = append(pings, filePings...)
		}
		return nil
	})

	return movements, pings, err
}

func processFile(filePath, fileName string) ([]MovementRecord, []PingRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var movements []MovementRecord
	var pings []PingRecord

	scanner := bufio.NewScanner(file)
	var currentMovementNumber string
	var inPingallSection bool

	// Regex patterns
	movementPattern := regexp.MustCompile(`\[node movements\]\s+(\d+):\s+move\s+(\w+):\s+moving\s+\w+\s+->\s+([0-9,-]+)`)
	pingallStartPattern := regexp.MustCompile(`\[pingall_full\]\s+(\d+):`)
	csvHeaderPattern := regexp.MustCompile(`^src,dst,tx,rx,loss_pct,avg_rtt_ms$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for node movement
		if matches := movementPattern.FindStringSubmatch(line); matches != nil {
			movement := MovementRecord{
				MovementNumber: matches[1],
				NodeName:       matches[2],
				Position:       matches[3],
				TestFile:       fileName,
			}
			movements = append(movements, movement)
			currentMovementNumber = matches[1]
			continue
		}

		// Check for pingall section start
		if matches := pingallStartPattern.FindStringSubmatch(line); matches != nil {
			currentMovementNumber = matches[1]
			inPingallSection = true
			continue
		}

		// Skip CSV header line
		if csvHeaderPattern.MatchString(line) {
			continue
		}

		// Process ping data lines
		if inPingallSection && strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 6 {
				// Clean up loss_pct: convert "+1 errors" to "100"
				lossPct := parts[4]
				if strings.Contains(lossPct, "+1 errors") {
					lossPct = "100"
				}

				// Clean up avg_rtt_ms: convert "?" to "0"
				avgRttMs := parts[5]
				if avgRttMs == "?" {
					avgRttMs = "0"
				}

				ping := PingRecord{
					MovementNumber: currentMovementNumber,
					TestFile:       fileName,
					Src:            parts[0],
					Dst:            parts[1],
					Tx:             parts[2],
					Rx:             parts[3],
					LossPct:        lossPct,
					AvgRttMs:       avgRttMs,
				}
				pings = append(pings, ping)
			}
		}

		// Reset pingall section when we hit an empty line or new section
		if line == "" || strings.HasPrefix(line, "[") {
			inPingallSection = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return movements, pings, nil
}

func writeToCSV(outputPath string, movements []MovementRecord, pings []PingRecord) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"data_type", "movement_number", "test_file", "node_name", "position",
		"src", "dst", "tx", "rx", "loss_pct", "avg_rtt_ms",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write movement records
	for _, movement := range movements {
		record := []string{
			"movement", movement.MovementNumber, movement.TestFile, movement.NodeName, movement.Position,
			"", "", "", "", "", "", // Empty ping fields
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	// Write ping records
	for _, ping := range pings {
		record := []string{
			"ping", ping.MovementNumber, ping.TestFile, "", "", // Empty movement fields
			ping.Src, ping.Dst, ping.Tx, ping.Rx, ping.LossPct, ping.AvgRttMs,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
