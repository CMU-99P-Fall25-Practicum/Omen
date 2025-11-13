package main

// This file exists because wails does not support anonymous structs so every sub-struct must be named.

//#region enums

type PropModel string

const (
	Friis              PropModel = "Friis"
	LogDistance        PropModel = "LogDistance"
	LogNormalShadowing PropModel = "LogNormalShadowing"
)

var AllPropModels = []struct {
	Value  PropModel
	TSName string
}{
	{Friis, "Friis"},
	{LogDistance, "LogDistance"},
	{LogNormalShadowing, "LogNormalShadowing"},
}

//#endregion enums

type PropagationModel struct {
	Model string  `json:"model"`
	Exp   float64 `json:"exp"`
	S     float64 `json:"s"`
}

type Nets struct {
	NoiseTh          int              `json:"noise_th"`
	PropagationModel PropagationModel `json:"propagation_model"`
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

type Topo struct {
	Nets     Nets  `json:"nets"`
	Aps      []AP  `json:"aps"`
	Stations []Sta `json:"stations"`
}

type Input struct {
	SchemaVersion string `json:"schemaVersion"`
	/*Meta          struct {
		Backend   string `json:"backend"`
		Name      string `json:"name"`
		DurationS int    `json:"duration_s"`
	} `json:"meta"`*/
	Topo Topo `json:"topo"`
	/*Tests []struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Timeframe int    `json:"timeframe"`
		Node      string `json:"node"`
		Position  string `json:"position"`
	} `json:"tests"`
	Username string `json:"username"`
	Password string `json:"password"`
	Address  string `json:"address"`*/
}
