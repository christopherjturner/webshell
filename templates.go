package main

import (
	"embed"
	"html/template"
)

//go:embed templates/*
var templateFS embed.FS

// As with assets, templates are embedded in the binary.
var (
	errorTemplate  = template.Must(template.ParseFS(templateFS, "templates/error.html"))
	fileTemplate   = template.Must(template.ParseFS(templateFS, "templates/files.html"))
	replayTemplate = template.Must(template.ParseFS(templateFS, "templates/replay.html"))
	termTemplate   = template.Must(template.ParseFS(templateFS, "templates/index.html"))
)
