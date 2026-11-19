package endpoint

import (
	"context"

	"example.com/devops-lab6/orders/order"
	kitendpoint "github.com/go-kit/kit/endpoint"
)

type CreateResponse struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Order order.Order `json:"order"`
}

type ChangeStatusRequest struct {
	ID     string
	Status string
}

type Endpoints struct {
	Create       kitendpoint.Endpoint
	GetByID      kitendpoint.Endpoint
	ChangeStatus kitendpoint.Endpoint
}

func MakeEndpoints(s order.Service) Endpoints {
	return Endpoints{
		Create: func(ctx context.Context, request any) (any, error) {
			id, err := s.Create(ctx, request.(order.Order))
			return CreateResponse{ID: id}, err
		},
		GetByID: func(ctx context.Context, request any) (any, error) {
			o, err := s.GetByID(ctx, request.(string))
			return GetResponse{Order: o}, err
		},
		ChangeStatus: func(ctx context.Context, request any) (any, error) {
			req := request.(ChangeStatusRequest)
			return struct{}{}, s.ChangeStatus(ctx, req.ID, req.Status)
		},
	}
}
