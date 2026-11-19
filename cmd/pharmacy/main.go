package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "example.com/devops-lab6/api/pharmacy/v1"
	"example.com/devops-lab6/internal/run"
	"example.com/devops-lab6/pharmacy/endpoint"
	"example.com/devops-lab6/pharmacy/implementation"
	"example.com/devops-lab6/pharmacy/repository"
	"example.com/devops-lab6/pharmacy/transport"
	"github.com/go-kit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	httpAddr := flag.String("http.addr", ":8080", "HTTP listen address")
	grpcAddr := flag.String("grpc.addr", ":9090", "gRPC listen address")
	flag.Parse()

	logger := log.With(log.NewSyncLogger(log.NewLogfmtLogger(os.Stderr)), "service", "pharmacy")
	if err := serve(*httpAddr, *grpcAddr, logger); err != nil {
		_ = logger.Log("err", err)
		os.Exit(1)
	}
}

func serve(httpAddr, grpcAddr string, logger log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return err
	}
	defer func() { _ = httpListener.Close() }()
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	defer func() { _ = grpcListener.Close() }()

	repo := repository.NewMemoryRepository()
	svc := implementation.NewService(repo)
	endpoints := endpoint.MakeEndpoints(svc)
	httpServer := run.HTTPServer(httpAddr, transport.NewHTTP(endpoints, logger))
	grpcServer := grpc.NewServer()
	pb.RegisterPharmacyServiceServer(grpcServer, transport.NewGRPC(endpoints))
	reflection.Register(grpcServer)

	done := make(chan error, 2)
	go func() { done <- httpServer.Serve(httpListener) }()
	go func() { done <- grpcServer.Serve(grpcListener) }()
	_ = logger.Log("transport", "HTTP", "addr", httpAddr)
	_ = logger.Log("transport", "gRPC", "addr", grpcAddr)

	var serveErr error
	select {
	case serveErr = <-done:
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), run.ShutdownTimeout)
	defer cancel()
	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()
	httpErr := run.ShutdownHTTP(shutdown, httpServer)
	select {
	case <-grpcStopped:
	case <-shutdown.Done():
		grpcServer.Stop()
	}
	if serveErr != nil {
		return serveErr
	}
	return httpErr
}
