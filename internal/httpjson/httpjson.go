package httpjson

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrBadJSON = errors.New("invalid JSON body")
var ErrTooLarge = errors.New("JSON body exceeds 1 MiB")

const maxBody = 1 << 20

func Decode(r *http.Request, value any) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return ErrBadJSON
	}
	if len(data) > maxBody {
		return ErrTooLarge
	}
	if err := json.Unmarshal(data, value); err != nil {
		return ErrBadJSON
	}
	return nil
}

func Encode(status int) func(context.Context, http.ResponseWriter, any) error {
	return func(_ context.Context, w http.ResponseWriter, value any) error {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		return json.NewEncoder(w).Encode(value)
	}
}

func Error(w http.ResponseWriter, err error, status int) {
	message := err.Error()
	switch {
	case errors.Is(err, ErrBadJSON):
		status = http.StatusBadRequest
	case errors.Is(err, ErrTooLarge):
		status = http.StatusRequestEntityTooLarge
	case status >= 500:
		message = "internal server error"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
