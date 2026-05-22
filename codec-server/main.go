// Package main runs a Temporal Codec Server that decodes (and encodes)
// payloads using the same AES-256-GCM EncryptionCodec the encryption worker
// registers. The Temporal Web UI calls /decode on this server to reveal
// plaintext for ciphertext payloads stored in workflow history.
//
// Browser-driven: the UI loads from http://localhost:8233 and calls this
// server cross-origin, so we wrap the SDK handler with a small CORS layer.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/converter"

	"github.com/alexandreroman/temporal-patterns-in-action/workers/encryption"
)

// maxRequestBytes caps incoming codec requests. Temporal payloads are small
// (history entries); 4 MiB leaves plenty of headroom while protecting the
// process from oversized POSTs.
const maxRequestBytes = 4 << 20

func main() {
	listen := getenv("LISTEN_ADDR", ":8888")
	uiOrigin := getenv("UI_ORIGIN", "http://localhost:8233")

	codec := &encryption.EncryptionCodec{Key: encryption.DemoKey}
	h, err := converter.NewPayloadHTTPHandler(converter.PayloadHTTPHandlerOptions{
		PreStorageCodecs: []converter.PayloadCodec{codec},
	})
	if err != nil {
		log.Fatalf("build payload handler: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", withCORS(uiOrigin, withBodyLimit(h)))

	srv := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("codec server listening on %s — UI origin %s", listen, uiOrigin)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("codec server stopped: %v", err)
	}
}

func withBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		next.ServeHTTP(w, r)
	})
}

func withCORS(allowed string, next http.Handler) http.Handler {
	origins := splitCSV(allowed)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowOrigin(origin, origins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			// Echo back whatever headers the preflight asks for. The Temporal
			// Web UI sends a moving set (x-namespace, caller-type, client-*,
			// temporal-namespace, …); listing them explicitly is brittle.
			if req := r.Header.Get("Access-Control-Request-Headers"); req != "" {
				w.Header().Set("Access-Control-Allow-Headers", req)
			} else {
				w.Header().Set("Access-Control-Allow-Headers",
					"Content-Type, Authorization, X-Namespace")
			}
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowOrigin(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
