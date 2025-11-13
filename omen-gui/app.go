package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rs/zerolog"
)

const outPath string = "in.json"

// App struct
type App struct {
	ctx    context.Context
	log    zerolog.Logger
	values Input
}

// NewApp creates a new App application struct
func NewApp() (*App, error) {
	l := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
	}).With().
		Timestamp().
		Caller().
		Logger().Level(zerolog.DebugLevel)
	return &App{log: l}, nil
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// AddAP attach an access point to the the input file.
func (a *App) AddAP(ap AP) {
	a.log.Info().Str("id", ap.ID).Msg("added access point")
	a.values.Topo.Aps = append(a.values.Topo.Aps, ap)
}

func (a *App) AddSta(sta Sta) {
	a.values.Topo.Stations = append(a.values.Topo.Stations, sta)
}

// GenerateJSON composes an input json from the current input values.
func (a *App) GenerateJSON() (success bool) {
	f, err := os.Create(outPath)
	if err != nil {
		a.log.Error().Err(err).Str("output path", outPath).Msg("failed to create output file")
		return false
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(a.values); err != nil {
		a.log.Error().Err(err).Str("output path", outPath).Msg("failed to encode values")
		return false
	}
	a.log.Info().Str("output path", outPath).Msg("successfully generated JSON")

	return true
}
