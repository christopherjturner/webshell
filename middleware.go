package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"time"
)

const sessionCookie = "cdpwebshell"

// Middleware to log inbound requests.
func requestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info(r.URL.Path)
		h.ServeHTTP(w, r)
	})
}

type Once struct {
	keyUsed    bool
	secretId   string
	cookiePath string
}

func NewOnceMiddleware(cookiePath string) *Once {
	return &Once{
		secretId:   generateId(),
		keyUsed:    false,
		cookiePath: cookiePath,
	}
}

// Allow only one connection and stop the server when its closed.
func (o *Once) once(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o.keyUsed {
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		}
		o.keyUsed = true
		h.ServeHTTP(w, r)
		os.Exit(0)
	})
}

func (o Once) requireCookie(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			logger.Error(err.Error())
			http.Error(w, "Access Denied", http.StatusUnauthorized)
			return
		}

		if o.secretId != cookie.Value {
			logger.Error("Cookie does not match session, unsetting cookie.")

			expire := &http.Cookie{
				Name:     sessionCookie,
				Value:    "",
				Path:     o.cookiePath,
				MaxAge:   0,
				HttpOnly: true,
			}
			http.SetCookie(w, expire)
			http.Error(w, "Access Denied", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// Sets a one-time cookie as part of the -once restrictions.
func (o *Once) setCookie(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !o.keyUsed {
			cookie := &http.Cookie{
				Name:     sessionCookie,
				Value:    o.secretId,
				Path:     o.cookiePath,
				Expires:  time.Now().Add(2 * time.Hour),
				HttpOnly: true,
			}
			http.SetCookie(w, cookie)
		}
		h.ServeHTTP(w, r)
	})
}

func generateId() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%X", b)
}
