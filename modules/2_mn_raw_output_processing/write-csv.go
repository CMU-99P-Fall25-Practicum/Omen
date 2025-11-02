package main

import (
	"encoding/csv"
	"os"
	"strings"

	"mn_raw_output_processing/models"
)

func writeToCSV(outputPath string, movements []models.MovementRecord, pings []models.PingRecord) error {
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

func writeIwToCSV(outputPath string, stations []models.StationRecord, aps []models.AccessPointRecord) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
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
		return err
	}

	// Write station records
	for _, station := range stations {
		record := []string{
			"station", station.TestFile, station.StationName, "", station.ConnectedTo, station.SSID,
			station.Freq, station.RXBytes, station.RXPackets, station.TXBytes, station.TXPackets,
			station.Signal, station.RxBitrate, station.TxBitrate, station.BssFlags, station.DtimPeriod,
			station.BeaconInt, "", "", "", "", "", "", "", "", "", "", "", "", "",
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	// Write AP records
	for _, ap := range aps {
		record := []string{
			"access_point", ap.TestFile, ap.APName, ap.Interface, "", "", "", ap.RXBytes, ap.RXPackets,
			ap.TXBytes, ap.TXPackets, "", "", "", "", "", "", ap.Flags, ap.MTU, ap.Ether,
			ap.TxQueueLen, ap.RXErrors, ap.RXDropped, ap.RXOverruns, ap.RXFrame, ap.TXErrors,
			ap.TXDropped, ap.TXOverruns, ap.TXCarrier, ap.TXCollisions,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func writeNodesCSV(outputPath string, nodes []models.NodeRecord) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"id", "title", "position", "rx_bytes", "rx_packets", "tx_bytes", "tx_packets", "success_pct_rate"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write node records
	for _, node := range nodes {
		record := []string{
			node.ID,
			node.Title,
			node.Position,
			node.RXBytes,
			node.RXPackets,
			node.TXBytes,
			node.TXPackets,
			node.SuccessPctRate,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func writeEdgesCSV(outputPath string, edges []models.EdgeRecord) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"id", "source", "target"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write edge records
	for _, edge := range edges {
		// Skip station-to-station communication
		if strings.HasPrefix(edge.Source, "sta") && strings.HasPrefix(edge.Target, "sta") {
			continue
		}
		record := []string{
			edge.ID,
			edge.Source,
			edge.Target,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
