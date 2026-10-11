package main

import (
	"embed"
	"log"

	"loadika-desktop/internal/app"
)

// Собранный фронтенд встраивается в бинарник и отдаётся webview.
// Подробнее: https://pkg.go.dev/embed
//
//go:embed all:frontend/dist
var assets embed.FS

// Иконка приложения встраивается в бинарник, чтобы окно/таскбар показывали её
// и при запуске без установки (на Linux — X11; на Wayland иконка берётся из .desktop).
//
//go:embed build/appicon.png
var appIcon []byte

func main() {
	if err := app.Run(assets, appIcon); err != nil {
		log.Fatal(err)
	}
}
