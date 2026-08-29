// Package main is the entry point for the Paradox Modding Tools application.
package main

import (
	"embed"
	"log"

	"paradox-modding-tools/services"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

//go:embed all:frontend/dist
var assets embed.FS

var version = "dev"

func main() {
	dbSvc := &services.DbService{}
	if err := dbSvc.ServiceStartup(); err != nil {
		log.Fatalf("db startup: %v", err)
	}

	fileSvc := &services.FileService{}
	mergeSvc := &services.MergeService{FileService: fileSvc}
	settingsSvc := &services.SettingsService{DB: dbSvc.DB, Version: version}
	workspaceSvc := &services.WorkspaceService{DB: dbSvc.DB}
	patcherSvc := &services.PatcherService{DB: dbSvc.DB, FileService: fileSvc, MergeService: mergeSvc}
	langModelSvc := &services.LanguageModelService{DB: dbSvc.DB}

	app := application.New(application.Options{
		Name:        "paradox-modding-tools",
		Description: "Desktop tools for Paradox Interactive game modding",
		Services: []application.Service{
			application.NewService(settingsSvc),
			application.NewService(fileSvc),
			application.NewService(workspaceSvc),
			application.NewService(langModelSvc),
			application.NewService(patcherSvc),
			application.NewService(mergeSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// DbService is internal (no RPC bindings), so wire its shutdown here.
	app.OnShutdown(func() {
		if err := dbSvc.ServiceShutdown(); err != nil {
			log.Printf("db shutdown: %v", err)
		}
	})

	gh, err := github.New(github.Config{
		Repository:   "idodavis/paradox-modding-tools",
		ChecksumAsset: "SHA256SUMS",
	})
	if err != nil {
		log.Fatalf("github.New: %v", err)
	}

	if err := app.Updater.Init(updater.Config{
		CurrentVersion: version,
		Providers:      []updater.Provider{gh},
	}); err != nil {
		log.Fatalf("updater.Init: %v", err)
	}

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

	if err := app.Run(); err != nil {
		log.Fatalf("app.Run: %v", err)
	}
}
