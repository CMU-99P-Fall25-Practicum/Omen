package main

// This file exists because wails does not support anonymous structs so every sub-struct must be named.

type PropagationModel struct {
	Model string  `json:"model"`
	Exp   float64 `json:"exp"`
	S     float64 `json:"s"`
}

type Nets struct {
	NoiseTh          int              `json:"noise_th"`
	PropagationModel PropagationModel `json:"propagation_model"`
}

type Topo struct {
	Nets Nets `json:"nets"`
	/*Aps []struct {
		ID       string `json:"id"`
		Mode     string `json:"mode"`
		Channel  int    `json:"channel"`
		Ssid     string `json:"ssid"`
		Position string `json:"position"`
	} `json:"aps"`
	Stations []struct {
		ID       string `json:"id"`
		Position string `json:"position"`
	} `json:"stations"`*/
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
