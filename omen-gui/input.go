package main

// This file exists because wails does not support anonymous structs so every sub-struct must be named.

//#region enums

// PropModel enumerates the three, supported propagation models mn-wifi supports
type PropModel string

const (
	Friis              PropModel = "friis"
	LogDistance        PropModel = "logDistance"
	LogNormalShadowing PropModel = "logNormalShadowing"
)

var AllPropModels = []struct {
	Value  PropModel
	TSName string
}{
	{Friis, "Friis"},
	{LogDistance, "LogDistance"},
	{LogNormalShadowing, "LogNormalShadowing"},
}

// WifiMode is as it says on the tin
type WifiMode string

const (
	A  WifiMode = "a"
	B  WifiMode = "b"
	G  WifiMode = "g"
	N  WifiMode = "n"
	AX WifiMode = "ax"
	AC WifiMode = "ac"
)

var AllWifiModes = []struct {
	Value  WifiMode
	TSName string
}{
	{A, "a"},
	{B, "b"},
	{G, "g"},
	{N, "n"},
	{AX, "ax"},
	{AC, "ac"},
}

//#endregion enums

type Meta struct {
	Backend   string `json:"backend"`
	Name      string `json:"name"`
	DurationS int    `json:"duration_s"`
}

//#region Topo and its children

type Topo struct {
	Nets     Nets  `json:"nets"`
	Aps      []AP  `json:"aps"`
	Stations []Sta `json:"stations"`
}

type Nets struct {
	NoiseTh          int              `json:"noise_th"`
	PropagationModel PropagationModel `json:"propagation_model"`
}

type PropagationModel struct {
	Model string  `json:"model"`
	Exp   float64 `json:"exp"`
	S     float64 `json:"s"`
}

type AP struct {
	ID       string `json:"id"`
	Mode     string `json:"mode"`
	Channel  int    `json:"channel"`
	SSID     string `json:"ssid"`
	Position string `json:"position"`
}

type Sta struct {
	ID       string `json:"id"`
	Position string `json:"position"`
}

//#endregion Topo and its children

type Test struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Timeframe int    `json:"timeframe"`
	Node      string `json:"node"`
	Position  string `json:"position"`
}

type Input struct {
	SchemaVersion string `json:"schemaVersion"`
	Meta          Meta   `json:"meta"`
	Topo          Topo   `json:"topo"`
	Tests         []Test `json:"tests"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Address       string `json:"address"`
}
