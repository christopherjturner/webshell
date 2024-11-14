package main

import (
	"fmt"
	"net/http"
)

type termPageParams struct {
	Token string
}

func termPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Printf("Token is being set to %s\n", config.Token)
	if err := termTemplate.Execute(w, termPageParams{Token: config.Token}); err != nil {
		logger.Error(fmt.Sprintf("%s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func replayPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := replayTemplate.Execute(w, termPageParams{Token: config.Token}); err != nil {
		logger.Error(fmt.Sprintf("%s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
