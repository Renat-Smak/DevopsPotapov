package repository

import (
	"context"
	"slices"
	"sync"

	"example.com/devops-lab6/orders/order"
)

type memory struct {
	mu     sync.RWMutex
	orders map[string]order.Order
}

func NewMemoryRepository() order.Repository {
	return &memory{orders: make(map[string]order.Order)}
}

func (r *memory) CreateOrder(_ context.Context, o order.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o.OrderItems = slices.Clone(o.OrderItems)
	r.orders[o.ID] = o
	return nil
}

func (r *memory) GetOrderByID(_ context.Context, id string) (order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return order.Order{}, order.ErrNotFound
	}
	o.OrderItems = slices.Clone(o.OrderItems)
	return o, nil
}

func (r *memory) ChangeOrderStatus(_ context.Context, id, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orders[id]
	if !ok {
		return order.ErrNotFound
	}
	o.Status = status
	r.orders[id] = o
	return nil
}
