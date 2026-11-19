package endpoint

import (
	"context"

	"example.com/devops-lab6/pharmacy/medicine"
	kitendpoint "github.com/go-kit/kit/endpoint"
)

type CreateResponse struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Medicine medicine.Medicine `json:"medicine"`
}

type SellRequest struct {
	ID    string
	Count int
}

type Endpoints struct {
	Create  kitendpoint.Endpoint
	GetByID kitendpoint.Endpoint
	Sell    kitendpoint.Endpoint
}

func MakeEndpoints(s medicine.Service) Endpoints {
	return Endpoints{
		Create: func(ctx context.Context, request any) (any, error) {
			id, err := s.Create(ctx, request.(medicine.Medicine))
			return CreateResponse{ID: id}, err
		},
		GetByID: func(ctx context.Context, request any) (any, error) {
			m, err := s.GetByID(ctx, request.(string))
			return GetResponse{Medicine: m}, err
		},
		Sell: func(ctx context.Context, request any) (any, error) {
			req := request.(SellRequest)
			return struct{}{}, s.Sell(ctx, req.ID, req.Count)
		},
	}
}
