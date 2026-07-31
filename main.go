package main

import (
	"embed"
	"log"

	"paradox-modding-tools/services"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

var version = "dev" // This will be replaced by the build process with the actual version number

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	// Main application initialization
	dbSvc := &services.DbService{}
	if err := dbSvc.ServiceStartup(); err != nil {
		log.Fatalf("db startup: %v", err)
	}

	logSvc := &services.LogService{}
	if err := logSvc.ServiceStartup(); err != nil {
		log.Fatalf("log startup: %v", err)
	}

	fileSvc := &services.FileService{}
	mergeSvc := &services.MergeService{FileService: fileSvc}
	modDocSvc := &services.ModDocService{FileService: fileSvc, DB: dbSvc.DB}
	settingsSvc := &services.SettingsService{DB: dbSvc.DB, Version: version}
	steamSvc := &services.SteamService{DB: dbSvc.DB}
	invSvc := &services.InventoryService{DB: dbSvc.DB}

	app := application.New(application.Options{
		Name:        "paradox-modding-tools",
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(logSvc),
			application.NewService(dbSvc),
			application.NewService(fileSvc),
			application.NewService(&services.BrowserService{}),
			application.NewService(&services.CompareService{}),
			application.NewService(modDocSvc),
			application.NewService(settingsSvc),
			application.NewService(steamSvc),
			application.NewService(&services.ClipboardService{}),
			application.NewService(invSvc),
			application.NewService(mergeSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Initialize the GitHub updater provider with the specified repository.
	// This allows the application to check for updates from the GitHub repository releases.
	// Using SHA256SUMS sidecar for checksum verification of downloaded updates.
	gh, err := github.New(github.Config{Repository: "idodavis/paradox-modding-tools", ChecksumAsset: "SHA256SUMS"})
	if err != nil {
		log.Fatalf("github.New: %v", err)
	}

	if err := app.Updater.Init(updater.Config{CurrentVersion: version, Providers: []updater.Provider{gh}}); err != nil {
		log.Fatalf("updater.Init: %v", err)
	}

	// Main App Window
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Paradox Modding Tools",
		Width:  1300,
		Height: 900,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		URL: "/",
	})

	// Run the application. This blocks until the application has been exited.
	if err := app.Run(); err != nil {
		log.Fatalf("app.Run: %v", err)
	}
}
