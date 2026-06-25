package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

var csrfKey []byte

const csrfKeyFile = ".csrf_key"

func initCSRF() error {
	data, err := os.ReadFile(csrfKeyFile)
	if err != nil || len(data) != 32 {
		csrfKey = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, csrfKey); err != nil {
			return err
		}
		if err := os.WriteFile(csrfKeyFile, csrfKey, 0600); err != nil {
			slog.Warn("could not save csrf key", "err", err)
		}
		slog.Info("generated new CSRF key")
	} else {
		csrfKey = data
		slog.Info("loaded existing CSRF key")
	}
	return nil
}

func csrfToken(sessionID string) string {
	mac := hmac.New(sha256.New, csrfKey)
	mac.Write([]byte(sessionID))
	mac.Write([]byte(time.Now().Format("2006-01-02")))
	return hex.EncodeToString(mac.Sum(nil))
}

func validateCSRF(r *http.Request) bool {
	if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
		return true
	}
	if r.URL.Path == "/login" {
		return true
	}
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	expected := csrfToken(sessionCookie.Value)

	token := r.FormValue("csrf_token")
	if token == "" {
		token = r.Header.Get("X-CSRF-Token")
	}
	if token == "" {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(token))
}

func isCSRFBypass(path string) bool {
	return strings.HasPrefix(path, "/api/chat/") || strings.HasPrefix(path, "/api/usuarios/") || strings.HasPrefix(path, "/api/tickets")
}

func withCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isCSRFBypass(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !validateCSRF(r) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, `{"error":"invalid csrf token"}`, http.StatusForbidden)
			} else {
				http.Error(w, "CSRF validation failed", http.StatusForbidden)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}
