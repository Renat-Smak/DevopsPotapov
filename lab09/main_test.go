package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestCounterAndHealth(t *testing.T) {
	h := newHandler(false)
	check := func(method, path string, status int, count *uint64) {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		if w.Code != status {
			t.Fatalf("%s %s: got HTTP %d, want %d", method, path, w.Code, status)
		}
		if count != nil {
			var result struct{ Count uint64 }
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Count != *count {
				t.Fatalf("%s: got %q, want count %d", path, w.Body, *count)
			}
		}
	}
	zero, one, two := uint64(0), uint64(1), uint64(2)
	check("GET", "/count", 200, &zero)
	check("GET", "/", 200, &one)
	check("GET", "/health", 200, nil)
	check("HEAD", "/", 200, nil)
	check("GET", "/count", 200, &one)
	check("GET", "/", 200, &two)
	check("GET", "/missing", 404, nil)

	w := httptest.NewRecorder()
	newHandler(true).ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != 500 {
		t.Fatalf("test failure mode: got HTTP %d, want 500", w.Code)
	}
}
