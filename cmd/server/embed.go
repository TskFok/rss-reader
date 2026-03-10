package main

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFS embed.FS

func getStaticFS() fs.FS {
	sub, _ := fs.Sub(staticFS, "static")
	return sub
}
