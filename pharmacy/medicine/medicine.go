package medicine

import (
	"context"
	"errors"
)

var (
	ErrNotFound          = errors.New("medicine not found")
	ErrInvalidMedicine   = errors.New("name and manufacturer are required; price and quantity must be non-negative")
	ErrInvalidCount      = errors.New("count must be positive")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidID         = errors.New("medicine id is required")
)

type Medicine struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	Price        float64 `json:"price"`
	Prescription bool    `json:"prescription"`
	Quantity     int     `json:"quantity"`
}

type Service interface {
	Create(ctx context.Context, medicine Medicine) (string, error)
	GetByID(ctx context.Context, id string) (Medicine, error)
	Sell(ctx context.Context, id string, count int) error
}

type Repository interface {
	Create(ctx context.Context, medicine Medicine) error
	GetByID(ctx context.Context, id string) (Medicine, error)
	Sell(ctx context.Context, id string, count int) error
}
