package main

import (
	"fmt"
	"net/http"
	"time"
)

type termPageParams struct {
	Token   string
	Title   string
	Start   int64
	Timeout int
}

func termPageHandler(token string, title string, start time.Time, timeout int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		params := termPageParams{
			Token: token,
			Title: title, Start: start.Unix() * 1000,
			Timeout: timeout,
		}
		if err := termTemplate.Execute(w, params); err != nil {
			logger.Error(fmt.Sprintf("%s", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}

func replayPageHandler(token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := replayTemplate.Execute(w, termPageParams{Token: token}); err != nil {
			logger.Error(fmt.Sprintf("%s", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}
