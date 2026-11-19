package transport

import (
	"context"
	"errors"
	"net/http"

	"example.com/devops-lab6/internal/httpjson"
	"example.com/devops-lab6/pharmacy/endpoint"
	"example.com/devops-lab6/pharmacy/medicine"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
)

func NewHTTP(e endpoint.Endpoints, logger log.Logger) http.Handler {
	router := mux.NewRouter()
	options := []kithttp.ServerOption{
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(encodeHTTPError),
	}
	router.Handle("/medicines", kithttp.NewServer(e.Create, decodeHTTPCreate, httpjson.Encode(http.StatusCreated), options...)).Methods(http.MethodPost)
	router.Handle("/medicines/{id}", kithttp.NewServer(e.GetByID, decodeHTTPID, httpjson.Encode(http.StatusOK), options...)).Methods(http.MethodGet)
	router.Handle("/medicines/{id}/sell", kithttp.NewServer(e.Sell, decodeHTTPSell, httpjson.Encode(http.StatusOK), options...)).Methods(http.MethodPost)
	return router
}

func decodeHTTPCreate(_ context.Context, r *http.Request) (any, error) {
	var m medicine.Medicine
	err := httpjson.Decode(r, &m)
	return m, err
}

func decodeHTTPID(_ context.Context, r *http.Request) (any, error) {
	return mux.Vars(r)["id"], nil
}

func decodeHTTPSell(_ context.Context, r *http.Request) (any, error) {
	var body struct {
		Count int `json:"count"`
	}
	err := httpjson.Decode(r, &body)
	return endpoint.SellRequest{ID: mux.Vars(r)["id"], Count: body.Count}, err
}

func encodeHTTPError(_ context.Context, err error, w http.ResponseWriter) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, medicine.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, medicine.ErrInvalidMedicine), errors.Is(err, medicine.ErrInvalidCount), errors.Is(err, medicine.ErrInvalidID):
		status = http.StatusBadRequest
	case errors.Is(err, medicine.ErrInsufficientStock):
		status = http.StatusConflict
	}
	httpjson.Error(w, err, status)
}
