package repository

import (
	"context"
	"sync"

	"example.com/devops-lab6/pharmacy/medicine"
)

type memory struct {
	mu        sync.RWMutex
	medicines map[string]medicine.Medicine
}

func NewMemoryRepository() medicine.Repository {
	return &memory{medicines: make(map[string]medicine.Medicine)}
}

func (r *memory) Create(_ context.Context, m medicine.Medicine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.medicines[m.ID] = m
	return nil
}

func (r *memory) GetByID(_ context.Context, id string) (medicine.Medicine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.medicines[id]
	if !ok {
		return medicine.Medicine{}, medicine.ErrNotFound
	}
	return m, nil
}

func (r *memory) Sell(_ context.Context, id string, count int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.medicines[id]
	if !ok {
		return medicine.ErrNotFound
	}
	if m.Quantity < count {
		return medicine.ErrInsufficientStock
	}
	m.Quantity -= count
	r.medicines[id] = m
	return nil
}
