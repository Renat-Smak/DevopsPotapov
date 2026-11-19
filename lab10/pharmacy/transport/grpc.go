package transport

import (
	"context"
	"errors"

	pb "example.com/devops-lab6/api/pharmacy/v1"
	"example.com/devops-lab6/pharmacy/endpoint"
	"example.com/devops-lab6/pharmacy/medicine"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	pb.UnimplementedPharmacyServiceServer
	create  kitgrpc.Handler
	getByID kitgrpc.Handler
	sell    kitgrpc.Handler
}

func NewGRPC(e endpoint.Endpoints) pb.PharmacyServiceServer {
	return &grpcServer{
		create:  kitgrpc.NewServer(e.Create, decodeGRPCCreate, encodeGRPCCreate),
		getByID: kitgrpc.NewServer(e.GetByID, decodeGRPCID, encodeGRPCGet),
		sell:    kitgrpc.NewServer(e.Sell, decodeGRPCSell, encodeGRPCSell),
	}
}

func (s *grpcServer) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
	_, response, err := s.create.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcError(err)
	}
	return response.(*pb.CreateResponse), nil
}

func (s *grpcServer) GetByID(ctx context.Context, req *pb.GetByIDRequest) (*pb.GetByIDResponse, error) {
	_, response, err := s.getByID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcError(err)
	}
	return response.(*pb.GetByIDResponse), nil
}

func (s *grpcServer) Sell(ctx context.Context, req *pb.SellRequest) (*pb.SellResponse, error) {
	_, response, err := s.sell.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcError(err)
	}
	return response.(*pb.SellResponse), nil
}

func decodeGRPCCreate(_ context.Context, request any) (any, error) {
	m := request.(*pb.CreateRequest).GetMedicine()
	if m == nil {
		return nil, status.Error(codes.InvalidArgument, "medicine is required")
	}
	quantity := int(m.GetQuantity())
	if int64(quantity) != m.GetQuantity() {
		return nil, status.Error(codes.InvalidArgument, "quantity is outside the int range")
	}
	return medicine.Medicine{
		Name: m.GetName(), Manufacturer: m.GetManufacturer(),
		Price: m.GetPrice(), Prescription: m.GetPrescription(), Quantity: quantity,
	}, nil
}

func decodeGRPCID(_ context.Context, request any) (any, error) {
	return request.(*pb.GetByIDRequest).GetId(), nil
}

func decodeGRPCSell(_ context.Context, request any) (any, error) {
	req := request.(*pb.SellRequest)
	count := int(req.GetCount())
	if int64(count) != req.GetCount() {
		return nil, status.Error(codes.InvalidArgument, "count is outside the int range")
	}
	return endpoint.SellRequest{ID: req.GetId(), Count: count}, nil
}

func encodeGRPCCreate(_ context.Context, response any) (any, error) {
	return &pb.CreateResponse{Id: response.(endpoint.CreateResponse).ID}, nil
}

func encodeGRPCGet(_ context.Context, response any) (any, error) {
	m := response.(endpoint.GetResponse).Medicine
	return &pb.GetByIDResponse{Medicine: &pb.Medicine{
		Id: m.ID, Name: m.Name, Manufacturer: m.Manufacturer,
		Price: m.Price, Prescription: m.Prescription, Quantity: int64(m.Quantity),
	}}, nil
}

func encodeGRPCSell(_ context.Context, _ any) (any, error) {
	return &pb.SellResponse{}, nil
}

func grpcError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	switch {
	case errors.Is(err, medicine.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, medicine.ErrInvalidMedicine), errors.Is(err, medicine.ErrInvalidCount), errors.Is(err, medicine.ErrInvalidID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, medicine.ErrInsufficientStock):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
