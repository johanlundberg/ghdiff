package web

import "embed"

//go:embed index.html css/* js/* vendor/*
var Assets embed.FS
