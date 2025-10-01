package models

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
