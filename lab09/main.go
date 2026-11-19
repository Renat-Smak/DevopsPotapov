package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

func newHandler(healthFail bool) http.Handler {
	var count atomic.Uint64
	mux := http.NewServeMux()
	writeCount := func(w http.ResponseWriter, value uint64) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Count uint64 `json:"count"`
		}{Count: value})
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeCount(w, count.Add(1))
	})
	mux.HandleFunc("GET /count", func(w http.ResponseWriter, _ *http.Request) {
		writeCount(w, count.Load())
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		if healthFail {
			http.Error(w, "health check failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func main() {
	server := &http.Server{
		Addr:              ":8080",
		Handler:           newHandler(os.Getenv("LAB9_HEALTH_FAIL") == "1"),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
