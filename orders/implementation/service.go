package implementation

import (
	"context"
	"strings"
	"time"

	"example.com/devops-lab6/orders/order"
	"github.com/google/uuid"
)

type service struct{ repo order.Repository }

func NewService(repo order.Repository) order.Service { return &service{repo: repo} }

func (s *service) Create(ctx context.Context, o order.Order) (string, error) {
	o.ID = uuid.NewString()
	o.Status = "Pending"
	o.CreatedOn = time.Now().Unix()
	if err := s.repo.CreateOrder(ctx, o); err != nil {
		return "", err
	}
	return o.ID, nil
}

func (s *service) GetByID(ctx context.Context, id string) (order.Order, error) {
	return s.repo.GetOrderByID(ctx, id)
}

func (s *service) ChangeStatus(ctx context.Context, id, status string) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return order.ErrInvalidStatus
	}
	return s.repo.ChangeOrderStatus(ctx, id, status)
}
