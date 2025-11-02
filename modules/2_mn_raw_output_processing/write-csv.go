package main

import (
	"encoding/csv"
	"os"
	"strconv"

	"Omen/modules/2_mn_raw_output_processing/models"
)

// writePingAllFull writes ping data from complete test to the given output.
//
// Uses the following format:
// data_type,movement_number,test_file,node_name,position,src,dst,tx,rx,loss_pct,avg_rtt_ms
//
// NOTE(rlandau): This format is somewhat a relic from earlier I/O Contracts.
// data_type is always "ping" and node_name+position are always empty.
func writePingAllFull(outputPath string, parsed []models.ParsedRawFile) (count uint, _ error) {
	file, err := os.Create(outputPath)
	if err != nil {
		return 0, err
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
		return 0, err
	}

	// collect ping data from all files
	for _, p := range parsed {
		for _, ping := range p.Pings {
			record := []string{
				"ping", ping.MovementNumber, ping.TestFile, "", "", // Empty movement fields
				ping.Src, ping.Dst, ping.Tx, ping.Rx, ping.LossPct, ping.AvgRttMs,
			}
			if err := writer.Write(record); err != nil {
				return count, err
			}
			count = +1
		}
	}

	return count, nil
}

// writeIWFull walks the parsed models and writes their connection information into the file at outputPath.
//
// The file will contain all stas from all raw files followed by all aps from all raw files.
func writeIWFull(outputPath string, parsed []models.ParsedRawFile) (staCount, apCount uint, _ error) {
	file, err := os.Create(outputPath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"device_type", "test_file", "device_name", "interface", "connected_to", "ssid", "freq",
		"rx_bytes", "rx_packets", "tx_bytes", "tx_packets", "signal", "rx_bitrate", "tx_bitrate",
		"bss_flags", "dtim_period", "beacon_int", "flags", "mtu", "ether", "tx_queue_len",
		"rx_errors", "rx_dropped", "rx_overruns", "rx_frame", "tx_errors", "tx_dropped",
		"tx_overruns", "tx_carrier", "tx_collisions",
	}
	if err := writer.Write(header); err != nil {
		return 0, 0, err
	}

	// Write station records
	for _, p := range parsed {
		for _, station := range p.Stations {
			record := []string{
				"station", station.TestFile, station.StationName, "", station.ConnectedTo, station.SSID,
				station.Freq, station.RXBytes, station.RXPackets, station.TXBytes, station.TXPackets,
				station.Signal, station.RxBitrate, station.TxBitrate, station.BssFlags, station.DtimPeriod,
				station.BeaconInt, "", "", "", "", "", "", "", "", "", "", "", "", "",
			}
			if err := writer.Write(record); err != nil {
				return staCount, apCount, err
			}
			staCount += 1
		}
	}
	// Write AP records
	for _, p := range parsed {
		for _, ap := range p.APs {
			record := []string{
				"access_point", ap.TestFile, ap.APName, ap.Interface, "", "", "", ap.RXBytes, ap.RXPackets,
				ap.TXBytes, ap.TXPackets, "", "", "", "", "", "", ap.Flags, ap.MTU, ap.Ether,
				ap.TxQueueLen, ap.RXErrors, ap.RXDropped, ap.RXOverruns, ap.RXFrame, ap.TXErrors,
				ap.TXDropped, ap.TXOverruns, ap.TXCarrier, ap.TXCollisions,
			}
			if err := writer.Write(record); err != nil {
				return staCount, apCount, err
			}
			apCount += 1
		}
	}

	return staCount, apCount, nil
}

// Params:
//
// outPath: the file path to create/truncate and write data to.
//
// timeframe: the timeframe we are processing for (under the "movement_number" column)
//
// rawTestFileName: "timeframeX.txt", where X==timeframe
func writeMovementCSV(outPath string, timeframe uint64, rawTestFileName string, pings []models.PingRecord) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	wr := csv.NewWriter(f)
	defer wr.Flush()

	// header
	hdr := []string{"data_type", "movement_number", "test_file", "node_name", "position", "src", "dst", "tx", "rx", "loss_pct", "avg_rtt_ms"}
	if err := wr.Write(hdr); err != nil {
		return err
	}

	// records
	record := []string{
		"ping", strconv.FormatUint(timeframe, 10),
		rawTestFileName,
		"", // node name is always empty
		"", // position is always empty
		//pings, // staX
		// staY
		// TODO
	}
	if err := wr.Write(record); err != nil {
		// TODO
	}
	return nil
}
