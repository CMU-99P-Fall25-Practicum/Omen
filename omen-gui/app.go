package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

const outPath string = "in.json"

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() (*App, error) {
	return &App{}, nil
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GenerateJSON composes an input json from the current input values.
func (a *App) GenerateJSON(values Input) (success bool) {
	f, err := os.Create(outPath)
	if err != nil {
		log.Error().Err(err).Str("output path", outPath).Msg("failed to create output file")
		return false
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(values); err != nil {
		log.Error().Err(err).Str("output path", outPath).Msg("failed to encode values")
		return false
	}
	return true
}
