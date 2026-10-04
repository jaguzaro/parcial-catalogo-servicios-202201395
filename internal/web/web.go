// Package web expone la interfaz compilada, embebida en el binario.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Archivos devuelve la interfaz con dist/ como raiz.
func Archivos() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // dist esta embebido en compilacion; no puede faltar
	}
	return sub
}
