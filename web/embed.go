// Package web contains embedded static assets for the frontend.
package web

import "embed"

//go:embed index.html css/* js/* vendor/*

// Assets contains the embedded frontend files.
var Assets embed.FS
