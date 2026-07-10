package main

import (
	"embed"
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/muhamm-ad/stratus/cmd/shared"
	"github.com/muhamm-ad/stratus/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	svc, warnings, err := shared.Init()
	if err != nil {
		panic(fmt.Sprintf("service initialization: %v", err))
	}
	for _, w := range warnings {
		fmt.Printf("provider unavailable: %v\n", w) // incomplete config → skipped
	}

	application := app.NewApp(svc)

	err = wails.Run(&options.App{
		Title:  "stratus",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        application.Startup,
		Bind: []interface{}{
			application,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
