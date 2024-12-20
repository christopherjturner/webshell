package main

import (
	"fmt"
	"net/http"
)

type termPageParams struct {
	Token string
	Title string
}

func termPageHandler(token string, title string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fmt.Printf("Token is being set to %s\n", token)
		if err := termTemplate.Execute(w, termPageParams{Token: token, Title: title}); err != nil {
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
