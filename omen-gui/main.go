/*
Package main provides the backend and driver functionality of the GUI.

This includes backend functions and struct definitions to-be bound to the front end.
*/
package main

// This file is basically just boilerplate code from Wails so it can spool itself up.

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := NewApp()
	if err != nil {
		panic(err)
	}

	// set the basic parameters and bound objects
	opts := options.App{
		Title:  "input generator",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 60, G: 60, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []any{
			app,
		},
		EnumBind: []any{
			AllPropModels,
			AllWifiModes,
		},
	}

	if err = wails.Run(&opts); err != nil {
		app.log.Error().Err(err).Msg("failed to run the app")
	}
}
