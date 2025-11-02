// Package models contains structs to serve as intermediate formats while transforming raw test output into well-formed data points to be visualized.
package models

// ParsedRawFile is the collection of records pulled a raw timeframeX.txt file.
// Each ParsedRawFile should represent exactly 1 timeframe.
type ParsedRawFile struct {
	Timeframe uint
	Path      string // file path
	Movements []MovementRecord
	Pings     []PingRecord
	Stations  []StationRecord
	APs       []AccessPointRecord
}

// A MovementRecord represents a single move action performed on a node during the last run.
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

type StationRecord struct {
	TestFile    string
	StationName string
	ConnectedTo string
	SSID        string
	Freq        string
	RXBytes     string
	RXPackets   string
	TXBytes     string
	TXPackets   string
	Signal      string
	RxBitrate   string
	TxBitrate   string
	BssFlags    string
	DtimPeriod  string
	BeaconInt   string
}

type AccessPointRecord struct {
	TestFile     string
	APName       string
	Interface    string
	Flags        string
	MTU          string
	Ether        string
	TxQueueLen   string
	RXPackets    string
	RXBytes      string
	RXErrors     string
	RXDropped    string
	RXOverruns   string
	RXFrame      string
	TXPackets    string
	TXBytes      string
	TXErrors     string
	TXDropped    string
	TXOverruns   string
	TXCarrier    string
	TXCollisions string
}

type NodeRecord struct {
	ID             string
	Title          string
	Position       string
	RXBytes        string
	RXPackets      string
	TXBytes        string
	TXPackets      string
	SuccessPctRate string
}

type EdgeRecord struct {
	ID     string
	Source string
	Target string
}
