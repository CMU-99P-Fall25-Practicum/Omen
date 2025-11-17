package main

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"slices"

	"github.com/rs/zerolog"
)

const outPath string = "in.json"

// App is the driver application itself.
// Some of the components of Input are stored in other fields for easier processing.
// Input is fully composed and marshaled in GenerateJSON.
type App struct {
	ctx context.Context
	log zerolog.Logger

	// input components

	values Input
	aps    map[string]AP  // ap name -> ap info
	sta    map[string]Sta // station name -> station info
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
	return &App{
		log: l,

		aps: map[string]AP{},
		sta: map[string]Sta{},
	}, nil
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// AddAP attach an access point to the the input file.
func (a *App) AddAP(ap AP) {
	// check if we are adding or editing
	_, found := a.aps[ap.ID]
	a.aps[ap.ID] = ap
	if !found { // add
		a.log.Info().Str("id", ap.ID).Msg("added access point")
	} else { // edit
		a.log.Info().Str("id", ap.ID).Msg("updated access point")
	}
}

func (a *App) AddSta(sta Sta) {
	// check if we are adding or editing
	_, found := a.sta[sta.ID]
	a.sta[sta.ID] = sta
	if !found { // add
		a.log.Info().Str("id", sta.ID).Msg("added station")
	} else { // edit
		a.log.Info().Str("id", sta.ID).Msg("updated station")
	}
}

// GenerateJSON composes an input json from the current input values.
func (a *App) GenerateJSON() (success bool) {
	f, err := os.Create(outPath)
	if err != nil {
		a.log.Error().Err(err).Str("output path", outPath).Msg("failed to create output file")
		return false
	}
	defer f.Close()

	// compose all values into struct
	a.values.Topo.Aps = slices.Collect(maps.Values(a.aps))
	a.values.Topo.Stations = slices.Collect(maps.Values(a.sta))

	a.log.Debug().Any("values", a.values).Msg("encoding values...")

	enc := json.NewEncoder(f)
	if err := enc.Encode(a.values); err != nil {
		a.log.Error().Err(err).Str("output path", outPath).Msg("failed to encode values")
		return false
	}
	a.log.Info().Str("output path", outPath).Msg("successfully generated JSON")

	return true
}
