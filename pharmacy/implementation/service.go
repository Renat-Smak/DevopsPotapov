package implementation

import (
	"context"
	"math"
	"strings"

	"example.com/devops-lab6/pharmacy/medicine"
	"github.com/google/uuid"
)

type service struct{ repo medicine.Repository }

func NewService(repo medicine.Repository) medicine.Service { return &service{repo: repo} }

func (s *service) Create(ctx context.Context, m medicine.Medicine) (string, error) {
	m.Name = strings.TrimSpace(m.Name)
	m.Manufacturer = strings.TrimSpace(m.Manufacturer)
	if m.Name == "" || m.Manufacturer == "" || m.Quantity < 0 ||
		m.Price < 0 || math.IsNaN(m.Price) || math.IsInf(m.Price, 0) {
		return "", medicine.ErrInvalidMedicine
	}
	m.ID = uuid.NewString()
	if err := s.repo.Create(ctx, m); err != nil {
		return "", err
	}
	return m.ID, nil
}

func (s *service) GetByID(ctx context.Context, id string) (medicine.Medicine, error) {
	if strings.TrimSpace(id) == "" {
		return medicine.Medicine{}, medicine.ErrInvalidID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) Sell(ctx context.Context, id string, count int) error {
	if strings.TrimSpace(id) == "" {
		return medicine.ErrInvalidID
	}
	if count <= 0 {
		return medicine.ErrInvalidCount
	}
	return s.repo.Sell(ctx, id, count)
}
