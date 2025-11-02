package main

import (
	"Omen/modules/2_mn_raw_output_processing/models"
	"bufio"
	"encoding/csv"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Regex patterns
// Updated to handle both old format (70,10,0) and new format ([70.0, 10.0, 0.0])
var (
	movementPattern     = regexp.MustCompile(`\[node movements\]\s+(\d+):\s+move\s+(\w+):\s+moving\s+\w+\s+->\s+\[?([0-9.,\s-]+)\]?`)
	pingallStartPattern = regexp.MustCompile(`\[pingall_full\]\s+(\d+):`)
	csvHeaderPattern    = regexp.MustCompile(`^src,dst,tx,rx,loss_pct,avg_rtt_ms$`)
	iwStartPattern      = regexp.MustCompile(`\[iw_stations\]`)
	stationPattern      = regexp.MustCompile(`^--- Station (\w+) ---$`)
	apPattern           = regexp.MustCompile(`^--- Access Point (\w+) ---$`)
)

// processRawFileDirectory processes each .txt file (expecting 1 file per timeframe, of the nomenclature 'timeframeX.txt') in the given directory,
// parsing the data into records for node movements, ping results, station info (via iw), and access point info (also via iw).
func processRawFileDirectory(directory string) ([]models.ParsedRawFile, error) {
	var parsed []models.ParsedRawFile

	err := filepath.WalkDir(directory, func(pth string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if d.IsDir() {
			return nil // continue
		}
		m := models.ParsedRawFile{
			Path: pth, // recombine path
		}
		if scanned, err := fmt.Sscanf(strings.ToLower(d.Name()), "timeframe%d.txt", &m.Timeframe); err != nil {
			return nil
		} else if scanned != 1 {
			return nil
		}
		fmt.Printf("Processing file: %s\n", m.Path)

		m.Movements, m.Pings, m.Stations, m.APs, err = processFile(pth, d.Name())
		if err != nil {
			fmt.Printf("Warning: Error processing file %s: %v\n", d.Name(), err)
			return nil // continue
		}
		// sanity check our index
		if len(parsed) != int(m.Timeframe) {
			fmt.Printf("Warning: parsed timeframe does not equal the current # of parsed models. %d parsed, %d latest timeframe", len(parsed), m.Timeframe)
		}

		parsed = append(parsed, m)
		return nil
	})

	return parsed, err
}

// processFile walks timeframeX.txt file to parse out usable data.
// Relies on direct string matches to figure out the structure of a line.
//
// If an error occurs, no arrays are returned to ensure incomplete data is not passed in.
func processFile(filePath, fileName string) (
	movements []models.MovementRecord, pings []models.PingRecord,
	stations []models.StationRecord, aps []models.AccessPointRecord,
	_ error,
) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer file.Close()

	var (
		currentMovementNumber string
		inPingallSection      bool
		inIwSection           bool
		currentStationName    string
		currentAPName         string
		inStationOutput       bool
		inAPOutput            bool
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for iw_stations section start
		if iwStartPattern.MatchString(line) {
			inIwSection = true
			continue
		}

		// Check for node movement
		if matches := movementPattern.FindStringSubmatch(line); matches != nil {
			movement := models.MovementRecord{
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

		// Process iw_stations data
		if inIwSection {
			// Check for station header
			if matches := stationPattern.FindStringSubmatch(line); matches != nil {
				currentStationName = matches[1]
				inStationOutput = false
				inAPOutput = false
				continue
			}

			// Check for AP header
			if matches := apPattern.FindStringSubmatch(line); matches != nil {
				currentAPName = matches[1]
				inStationOutput = false
				inAPOutput = false
				continue
			}

			// Check for Output: line
			if strings.HasPrefix(line, "Output:") {
				if currentStationName != "" {
					inStationOutput = true
				} else if currentAPName != "" {
					inAPOutput = true
				}
				continue
			}

			// Process station data
			if inStationOutput && currentStationName != "" {
				stations = processStationData(stations, line, currentStationName, fileName)
			}

			// Process AP data
			if inAPOutput && currentAPName != "" {
				aps = processAPData(aps, line, currentAPName, fileName)
			}

			// Reset when we hit a new section or end (station/AP header)
			if line == "" || strings.HasPrefix(line, "---") {
				// Before resetting, check if we have a station that wasn't added yet
				// (this happens when station is "Not connected")
				if inStationOutput && currentStationName != "" && !stationExists(stations, currentStationName, fileName) {
					// Create an empty station record for "Not connected" stations
					station := models.StationRecord{
						TestFile:    fileName,
						StationName: currentStationName,
					}
					stations = append(stations, station)
				}

				inStationOutput = false
				inAPOutput = false
				currentStationName = ""
				currentAPName = ""
			}
		}

		// Process ping data lines
		if inPingallSection && strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 6 {
				src := parts[0]
				dst := parts[1]

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

				ping := models.PingRecord{
					MovementNumber: currentMovementNumber,
					TestFile:       fileName,
					Src:            src,
					Dst:            dst,
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
		return nil, nil, nil, nil, err
	}

	return movements, pings, stations, aps, nil
}

func processStationData(stations []models.StationRecord, line, stationName, fileName string) []models.StationRecord {
	line = strings.TrimSpace(line)

	// Check if this is the start of a new station record
	if strings.HasPrefix(line, "Connected to ") {
		// Extract MAC address
		connectedPattern := regexp.MustCompile(`^Connected to ([0-9a-f:]+)`)
		if matches := connectedPattern.FindStringSubmatch(line); matches != nil {
			station := models.StationRecord{
				TestFile:    fileName,
				StationName: stationName,
				ConnectedTo: matches[1],
			}
			stations = append(stations, station)
		}
	} else if len(stations) > 0 {
		// Update the last station record with additional data
		lastIdx := len(stations) - 1
		if stations[lastIdx].StationName == stationName {
			updateStationField(&stations[lastIdx], line)
		}
	}

	return stations
}

func updateStationField(station *models.StationRecord, line string) {
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "SSID: ") {
		station.SSID = strings.TrimPrefix(line, "SSID: ")
	} else if strings.HasPrefix(line, "freq: ") {
		station.Freq = strings.TrimPrefix(line, "freq: ")
	} else if strings.HasPrefix(line, "RX: ") {
		// Extract bytes and packets from "RX: 343809 bytes (8714 packets)"
		rxPattern := regexp.MustCompile(`RX: (\d+) bytes \((\d+) packets\)`)
		if matches := rxPattern.FindStringSubmatch(line); matches != nil {
			station.RXBytes = matches[1]
			station.RXPackets = matches[2]
		}
	} else if strings.HasPrefix(line, "TX: ") {
		// Extract bytes and packets from "TX: 4898 bytes (68 packets)"
		txPattern := regexp.MustCompile(`TX: (\d+) bytes \((\d+) packets\)`)
		if matches := txPattern.FindStringSubmatch(line); matches != nil {
			station.TXBytes = matches[1]
			station.TXPackets = matches[2]
		}
	} else if strings.HasPrefix(line, "signal: ") {
		station.Signal = strings.TrimPrefix(line, "signal: ")
	} else if strings.HasPrefix(line, "rx bitrate: ") {
		station.RxBitrate = strings.TrimPrefix(line, "rx bitrate: ")
	} else if strings.HasPrefix(line, "tx bitrate: ") {
		station.TxBitrate = strings.TrimPrefix(line, "tx bitrate: ")
	} else if strings.HasPrefix(line, "bss flags: ") {
		station.BssFlags = strings.TrimPrefix(line, "bss flags: ")
	} else if strings.HasPrefix(line, "dtim period: ") {
		station.DtimPeriod = strings.TrimPrefix(line, "dtim period: ")
	} else if strings.HasPrefix(line, "beacon int: ") {
		station.BeaconInt = strings.TrimPrefix(line, "beacon int: ")
	}
}

func processAPData(aps []models.AccessPointRecord, line, apName, fileName string) []models.AccessPointRecord {
	line = strings.TrimSpace(line)

	// Check if this is the interface line (start of AP record)
	if strings.Contains(line, ": flags=") {
		// Extract interface name and basic info
		parts := strings.Split(line, ":")
		if len(parts) > 0 {
			interfaceName := strings.TrimSpace(parts[0])

			ap := models.AccessPointRecord{
				TestFile:  fileName,
				APName:    apName,
				Interface: interfaceName,
			}

			// Extract flags, MTU, etc. from the line
			updateAPField(&ap, line)
			aps = append(aps, ap)
		}
	} else if len(aps) > 0 {
		// Update the last AP record with additional data
		lastIdx := len(aps) - 1
		if aps[lastIdx].APName == apName {
			updateAPField(&aps[lastIdx], line)
		}
	}

	return aps
}

func updateAPField(ap *models.AccessPointRecord, line string) {
	line = strings.TrimSpace(line)

	// Parse the main interface line
	if strings.Contains(line, "flags=") && strings.Contains(line, "mtu") {
		// Extract flags pattern
		flagsPattern := regexp.MustCompile(`flags=(\d+)<([^>]+)>`)
		if matches := flagsPattern.FindStringSubmatch(line); matches != nil {
			ap.Flags = matches[2]
		}

		// Extract MTU
		mtuPattern := regexp.MustCompile(`mtu (\d+)`)
		if matches := mtuPattern.FindStringSubmatch(line); matches != nil {
			ap.MTU = matches[1]
		}

		// Extract txqueuelen
		txqPattern := regexp.MustCompile(`txqueuelen (\d+)`)
		if matches := txqPattern.FindStringSubmatch(line); matches != nil {
			ap.TxQueueLen = matches[1]
		}
	} else if strings.HasPrefix(line, "ether ") {
		etherPattern := regexp.MustCompile(`ether ([0-9a-f:]+)`)
		if matches := etherPattern.FindStringSubmatch(line); matches != nil {
			ap.Ether = matches[1]
		}
	} else if strings.HasPrefix(line, "RX packets") {
		// Parse "RX packets 137  bytes 8598 (8.5 KB)"
		rxPattern := regexp.MustCompile(`RX packets (\d+)\s+bytes (\d+)`)
		if matches := rxPattern.FindStringSubmatch(line); matches != nil {
			ap.RXPackets = matches[1]
			ap.RXBytes = matches[2]
		}
	} else if strings.HasPrefix(line, "RX errors") {
		// Parse "RX errors 0  dropped 0  overruns 0  frame 0"
		rxErrPattern := regexp.MustCompile(`RX errors (\d+)\s+dropped (\d+)\s+overruns (\d+)\s+frame (\d+)`)
		if matches := rxErrPattern.FindStringSubmatch(line); matches != nil {
			ap.RXErrors = matches[1]
			ap.RXDropped = matches[2]
			ap.RXOverruns = matches[3]
			ap.RXFrame = matches[4]
		}
	} else if strings.HasPrefix(line, "TX packets") {
		// Parse "TX packets 137  bytes 11064 (11.0 KB)"
		txPattern := regexp.MustCompile(`TX packets (\d+)\s+bytes (\d+)`)
		if matches := txPattern.FindStringSubmatch(line); matches != nil {
			ap.TXPackets = matches[1]
			ap.TXBytes = matches[2]
		}
	} else if strings.HasPrefix(line, "TX errors") {
		// Parse "TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0"
		txErrPattern := regexp.MustCompile(`TX errors (\d+)\s+dropped (\d+)\s+overruns (\d+)\s+carrier (\d+)\s+collisions (\d+)`)
		if matches := txErrPattern.FindStringSubmatch(line); matches != nil {
			ap.TXErrors = matches[1]
			ap.TXDropped = matches[2]
			ap.TXOverruns = matches[3]
			ap.TXCarrier = matches[4]
			ap.TXCollisions = matches[5]
		}
	}
}

// writeNodesCSV generates a nodes.csv file inside of tfDirPath using the parsed data for this timeframe.
func writeNodesCSV(parsed models.ParsedRawFile, tfDirPath string) error {
	// Calculate success rates based on cumulative pings
	successRates := calculateSuccessRates(parsed.Pings)

	// prep output file
	csvPath := path.Join(tfDirPath, "nodes.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// write header
	hdr := []string{"id", "title", "position", "rx_bytes", "rx_packets", "tx_bytes", "tx_packets", "success_pct_rate"}
	if err := writer.Write(hdr); err != nil {
		return err
	}

	// write stations
	for i, sta := range parsed.Stations {
		// validate that movement node lines up with station node
		if parsed.Movements[i].NodeName != sta.StationName {
			fmt.Printf("WARNING: movement node name does not match station name! node: %s != station: %s\n", parsed.Movements[i].NodeName, sta.StationName)
			continue
		}

		record := []string{
			sta.StationName,              // id
			sta.StationName,              // title
			parsed.Movements[i].Position, // position
			sta.RXBytes,
			sta.RXPackets,
			sta.TXBytes,
			sta.TXPackets,
			fmt.Sprintf("%.2f", successRates[sta.StationName]),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	// write aps
	for i, ap := range parsed.APs {
		// validate that movement node lines up with station node
		if parsed.Movements[i+len(parsed.Stations)].NodeName != ap.APName {
			fmt.Printf("WARNING: movement node name does not match station name! node: %s != station: %s\n", parsed.Movements[i].NodeName, ap.APName)
			continue
		}

		record := []string{
			ap.APName,
			ap.APName,
			parsed.Movements[i].Position,
			ap.RXBytes,
			ap.RXPackets,
			ap.TXBytes,
			ap.TXPackets,
			fmt.Sprintf("%.2f", successRates[ap.APName]),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	fmt.Printf("  Nodes CSV for timeframe %d written to: %s\n", parsed.Timeframe, csvPath)

	return nil
}

func writeEdgesCSV(parsed models.ParsedRawFile, tfDirPath string) error {
	// prep output file
	csvPath := path.Join(tfDirPath, "edges.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// write header
	header := []string{"id", "source", "target"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// use a map to consolidate duplicates; the map keys are broken apart later
	edges := map[string]struct {
		src    string
		target string
	}{}
	for _, ping := range parsed.Pings {
		// ignore station to station edges
		if strings.Contains(ping.Src, "sta") && strings.Contains(ping.Dst, "sta") {
			continue
		}

		id := ping.Src + "-" + ping.Dst
		edges[id] = struct {
			src    string
			target string
		}{
			ping.Src, ping.Dst,
		}
	}

	// sort and write the map into a file, breaking id into source and target
	elems := slices.Sorted(maps.Keys(edges))
	for _, id := range elems {
		record := []string{
			id,               // id
			edges[id].src,    // src
			edges[id].target, // target
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write line '%s' to %s: %w", id, csvPath, err)
		}
	}

	fmt.Printf("  Edges CSV for timeframe %d written to: %s\n", parsed.Timeframe, csvPath)

	return nil
}

func calculateSuccessRates(pings []models.PingRecord) map[string]float64 {
	successRates := make(map[string]float64)
	nodeCounts := make(map[string]int)
	nodeSuccesses := make(map[string]int)

	for _, ping := range pings {
		// Count for destination node
		nodeCounts[ping.Dst]++
		if ping.LossPct == "0" {
			nodeSuccesses[ping.Dst]++
		}

		// Count for source node
		nodeCounts[ping.Src]++
		if ping.LossPct == "0" {
			nodeSuccesses[ping.Src]++
		}
	}

	// Calculate success rates
	for node, totalCount := range nodeCounts {
		if totalCount > 0 {
			successRates[node] = (float64(nodeSuccesses[node]) / float64(totalCount))
		} else {
			successRates[node] = 0.0
		}
	}

	return successRates
}

// getCumulativePings returns all pings from test files up to and including the specified test file.
// The ordering is based on movement numbers extracted from the file names (e.g., test1.txt -> 1, test2.txt -> 2).
func getCumulativePings(allPings []models.PingRecord, upToTestFile string) []models.PingRecord {
	// Extract movement number from the target test file
	targetMovementNum := extractMovementNumber(upToTestFile)

	var cumulativePings []models.PingRecord
	for _, ping := range allPings {
		pingMovementNum := extractMovementNumber(ping.TestFile)
		if pingMovementNum <= targetMovementNum {
			cumulativePings = append(cumulativePings, ping)
		}
	}

	return cumulativePings
}

// getTestName extracts the test name from a test file name (e.g., "test1.txt" -> "test1")
func getTestName(testFile string) string {
	// Remove the .txt extension
	name := strings.TrimSuffix(testFile, ".txt")
	return name
}

// extractMovementNumber extracts the movement number from a test file name.
// For example, "test1.txt" -> 1, "test2.txt" -> 2, etc.
func extractMovementNumber(testFile string) int {
	// Extract the test name without extension
	name := getTestName(testFile)

	// Extract the number from the test name (e.g., "test1" -> 1)
	// This assumes the format is "testN" where N is a number
	numStr := strings.TrimPrefix(name, "test")

	// Try to parse the number
	var num int
	fmt.Sscanf(numStr, "%d", &num)
	return num
}

// stationExists checks if a station record already exists for the given station name and test file.
func stationExists(stations []models.StationRecord, stationName, testFile string) bool {
	for _, station := range stations {
		if station.StationName == stationName && station.TestFile == testFile {
			return true
		}
	}
	return false
}

// getPositionMap builds a map of node names to their positions from movement records.
// It returns the position for nodes in the specified test file.
func getPositionMap(movements []models.MovementRecord, testFile string) map[string]string {
	positionMap := make(map[string]string)

	// Get all movements from this specific test file
	for _, movement := range movements {
		if movement.TestFile == testFile {
			positionMap[movement.NodeName] = movement.Position
		}
	}

	return positionMap
}
