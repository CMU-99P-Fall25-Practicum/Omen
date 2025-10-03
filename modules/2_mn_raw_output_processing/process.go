package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mn_raw_output_processing/models"
)

func readFile(directory string) ([]models.MovementRecord, []models.PingRecord, []models.StationRecord, []models.AccessPointRecord, error) {
	var movements []models.MovementRecord
	var pings []models.PingRecord
	var stations []models.StationRecord
	var aps []models.AccessPointRecord

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".txt") {
			fmt.Printf("Processing file: %s\n", d.Name())

			fileMovements, filePings, fileStations, fileAPs, err := processFile(path, d.Name())
			if err != nil {
				fmt.Printf("Warning: Error processing file %s: %v\n", d.Name(), err)
				return nil // Continue with other files
			}

			movements = append(movements, fileMovements...)
			pings = append(pings, filePings...)
			stations = append(stations, fileStations...)
			aps = append(aps, fileAPs...)
		}
		return nil
	})

	return movements, pings, stations, aps, err
}

func processFile(filePath, fileName string) ([]models.MovementRecord, []models.PingRecord, []models.StationRecord, []models.AccessPointRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer file.Close()

	var movements []models.MovementRecord
	var pings []models.PingRecord
	var stations []models.StationRecord
	var aps []models.AccessPointRecord

	scanner := bufio.NewScanner(file)
	var currentMovementNumber string
	var inPingallSection bool
	var inIwSection bool
	var currentStationName string
	var currentAPName string
	var inStationOutput bool
	var inAPOutput bool

	// Regex patterns
	movementPattern := regexp.MustCompile(`\[node movements\]\s+(\d+):\s+move\s+(\w+):\s+moving\s+\w+\s+->\s+([0-9,-]+)`)
	pingallStartPattern := regexp.MustCompile(`\[pingall_full\]\s+(\d+):`)
	csvHeaderPattern := regexp.MustCompile(`^src,dst,tx,rx,loss_pct,avg_rtt_ms$`)
	iwStartPattern := regexp.MustCompile(`\[iw_stations\]`)
	stationPattern := regexp.MustCompile(`^--- Station (\w+) ---$`)
	apPattern := regexp.MustCompile(`^--- Access Point (\w+) ---$`)

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

			// Reset when we hit a new section or end
			if line == "" {
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

				// Skip station-to-station communication
				if strings.HasPrefix(src, "sta") && strings.HasPrefix(dst, "sta") {
					continue
				}

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
