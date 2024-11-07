package main

import (
	"net/http"
	"runtime"
)

func debugHandler(w http.ResponseWriter, r *http.Request) {
	buf := make([]byte, 1<<16) // 64 KB buffer
	l := runtime.Stack(buf, true)

	w.Write(buf[:l])
}
