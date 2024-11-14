package main

import (
	"embed"
	"html/template"
)

//go:embed templates/*
var templateFS embed.FS

// As with assets, templates are embedded in the binary.
var (
	termTemplate   = template.Must(template.ParseFS(templateFS, "templates/index.html"))
	replayTemplate = template.Must(template.ParseFS(templateFS, "templates/replay.html"))
	fileTemplate   = template.Must(template.ParseFS(templateFS, "templates/files.html"))
)
