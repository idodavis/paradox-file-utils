package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Paradox Script Merger",
		Width:     1300,
		Height:    1000,
		Assets:    assets,
		OnStartup: app.OnStartup,
		Bind:      []interface{}{app},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
