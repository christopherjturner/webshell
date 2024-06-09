package main

import (
	_ "embed"
	"fmt"
	"net/http"
)

// go:embed ./xterm/xterm.min.js
var xtermJs string

// go:embed ./xterm.min.css
var xtermCss string

func handleXtermJS(w http.ResponseWriter, r *http.Request) {
	println(xtermJs)
	w.Header().Set("Content-Type", "text/javascript")
	fmt.Printf("%v\n", r.URL)
	fmt.Fprint(w, xtermJs)
}
