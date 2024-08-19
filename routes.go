package main

import (
	"net/url"
	"path"
)

type Routes struct {
	Main    string
	Shell   string
	Home    string
	Upload  string
	GetFile string
	Assets  string
	Prefix  string
}

func BuildRoutes(token string) Routes {

	prefix := ""
	if token != "" {
		prefix = path.Join("/", url.PathEscape(token))
	}

	routes := Routes{
		Main:    path.Clean(path.Join(prefix, "/{$}")),
		Shell:   path.Clean(path.Join(prefix, "/shell")),
		Home:    path.Clean(path.Join(prefix, "/home")),
		Upload:  path.Clean(path.Join(prefix, "/upload")),
		GetFile: path.Clean(path.Join(prefix, "/home/{filename...}")),
		Assets:  path.Clean(path.Join(prefix, "/assets")) + "/",
		Prefix:  prefix,
	}

	return routes
}
