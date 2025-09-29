package models

/*
Input JSON Structure:
Input
├── schemaVersion (string)
├── meta
│   ├── backend (string)
│   ├── name (string)
│   └── duration_s (int)
├── topo
│   ├── host []
│   │   ├── id (string)
│   │   ├── tx_dbm (int, optional)
│   │   └── rx_sensitivity_dbm (int, optional)
│   ├── switch []
│   │   ├── id (string)
│   │   ├── tx_dbm (int, optional)
│   │   └── rx_sensitivity_dbm (int, optional)
│   ├── ap []
│   │   ├── id (string)
│   │   ├── tx_dbm (int, optional)
│   │   └── rx_sensitivity_dbm (int, optional)
│   └── links []
│       ├── node_id_a (string)
│       ├── node_id_b (string)
│       └── constraints (optional)
│           ├── loss_pkt (float64, optional)
│           ├── throughput_mbps (int, optional)
│           ├── mtu (int, optional)
│           └── delay_ms (int, optional)
├── tests []
│   ├── name (string)
│   ├── type (string)
│   ├── src (string)
│   ├── dst (string)
│   ├── count (int, optional)
│   ├── deadline_s (int, optional)
│   ├── duration_s (int, optional)
│   └── rate_mbps (int, optional)
├── username (string, optional)
├── password (string, optional)
└── address (string, optional)

*/

import (
	"net/netip"
)

// Main input structure that matches your new JSON format
type Input struct {
	SchemaVersion string `json:"schemaVersion"`
	Meta          Meta   `json:"meta"`
	Topo          Topo   `json:"topo"`
	Tests         []Test `json:"tests"`
	// Optional connection info in JSON
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	AP       string `json:"address,omitempty"`
}

// Meta information about the configuration
type Meta struct {
	Backend   string `json:"backend"`
	Name      string `json:"name"`
	DurationS int    `json:"duration_s"`
}

type Topo struct {
	Hosts    []Node `json:"hosts"`
	Switches []Node `json:"switches"`
	Aps      []Node `json:"aps"`
	Stations []Node `json:"stations"`
	Nets     Nets   `json:"nets"`
	Links    []Link `json:"links"`
}

type Nets struct {
	NoiseThreashold  int       `json:"noise_th"`
	PropagationModel Propmodel `json:"propagation_model"`
}

type Propmodel struct {
	Model string `json:"model"`
	Exp   int    `json:"exp"`
}

// Node represents a network node (host, switch, access point, etc.)
// (topo -> nodes)
type Node struct {
	ID               string `json:"id"`
	TxDBM            int    `json:"tx_dbm,omitempty"`
	RxSensitivityDBM int    `json:"rx_sensitivity_dbm,omitempty"`
	// WiFi-specific fields
	Mode     string `json:"mode,omitempty"`     // for APs
	Channel  int    `json:"channel,omitempty"`  // for APs
	SSID     string `json:"ssid,omitempty"`     // for APs
	Position string `json:"position,omitempty"` // for APs and stations
}

// Link represents a connection between two nodes (topo -> links)
type Link struct {
	NodeIDA     string      `json:"node_id_a"`
	NodeIDB     string      `json:"node_id_b"`
	Constraints Constraints `json:"constraints,omitempty"`
}

// Constraints for link properties (topo -> links -> constraints)
type Constraints struct {
	LossPkt        float64 `json:"loss_pkt,omitempty"`
	ThroughputMbps int     `json:"throughput_mbps,omitempty"`
	MTU            int     `json:"mtu,omitempty"`
	DelayMS        int     `json:"delay_ms,omitempty"`
}

// Test represents a network test to be performed
type Test struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Src       string `json:"src,omitempty"`
	Dst       string `json:"dst,omitempty"`
	Count     int    `json:"count,omitempty"`
	DeadlineS int    `json:"deadline_s,omitempty"`
	DurationS int    `json:"duration_s,omitempty"`
	RateMbps  int    `json:"rate_mbps,omitempty"`
	MoveNode  string `json:"node,omitempty"`     // MoveNode is the ID of the node to move (for "node movements" test type)
	Position  string `json:"position,omitempty"` // Position is a string representing coordinates, e.g., "x,y,z"
	CMD       string `json:"cmd,omitempty"`      // CMD is the command to run (for "iw" test type)
}

// Input Config from user to setup ssh connection to VM
type Config struct {
	Host             netip.AddrPort
	Username         string
	Password         string
	TopoFile         string
	UseCLI           bool
	RemotePathPython string
	RemotePathJSON   string
}
