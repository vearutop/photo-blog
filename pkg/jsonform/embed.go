package jsonform

import (
	"embed"

	"github.com/vearutop/statigz"
)

var (
	// FS holds embedded static assets.
	//
	//go:embed static/*
	staticAssets embed.FS

	staticServer = statigz.FileServer(staticAssets, statigz.FSPrefix("static"))
)
