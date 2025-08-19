package main

import (
	"fmt"
	"net/http"
	"runtime"
)

func debugHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("!!  CONN COUNT %d !!\n", connCount)
	buf := make([]byte, 1<<16) // 64 KB buffer
	l := runtime.Stack(buf, true)
	w.Write(buf[:l])
}

type SessionDebug struct {
	manager *SessionManager
}

func (sd SessionDebug) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for id, session := range sd.manager.sessions {
		w.Write([]byte(fmt.Sprintf("%s\t%v", id, session.lastActive)))
	}
}
