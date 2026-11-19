package transport

import (
	"context"
	"errors"
	"net/http"

	"example.com/devops-lab6/internal/httpjson"
	"example.com/devops-lab6/orders/endpoint"
	"example.com/devops-lab6/orders/order"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
)

func NewHTTP(e endpoint.Endpoints, logger log.Logger) http.Handler {
	router := mux.NewRouter()
	options := []kithttp.ServerOption{
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(encodeError),
	}
	router.Handle("/orders", kithttp.NewServer(e.Create, decodeCreate, httpjson.Encode(http.StatusCreated), options...)).Methods(http.MethodPost)
	router.Handle("/orders/{id}", kithttp.NewServer(e.GetByID, decodeID, httpjson.Encode(http.StatusOK), options...)).Methods(http.MethodGet)
	router.Handle("/orders/{id}/status", kithttp.NewServer(e.ChangeStatus, decodeStatus, httpjson.Encode(http.StatusOK), options...)).Methods(http.MethodPatch)
	return router
}

func decodeCreate(_ context.Context, r *http.Request) (any, error) {
	var o order.Order
	err := httpjson.Decode(r, &o)
	return o, err
}

func decodeID(_ context.Context, r *http.Request) (any, error) {
	return mux.Vars(r)["id"], nil
}

func decodeStatus(_ context.Context, r *http.Request) (any, error) {
	var body struct {
		Status string `json:"status"`
	}
	err := httpjson.Decode(r, &body)
	return endpoint.ChangeStatusRequest{ID: mux.Vars(r)["id"], Status: body.Status}, err
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, order.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, order.ErrInvalidStatus):
		status = http.StatusBadRequest
	}
	httpjson.Error(w, err, status)
}
