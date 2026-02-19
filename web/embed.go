package web

import "embed"

//go:embed index.html css js
var Assets embed.FS
