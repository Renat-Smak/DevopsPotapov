package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/devops-lab6/internal/run"
	"example.com/devops-lab6/orders/endpoint"
	"example.com/devops-lab6/orders/implementation"
	"example.com/devops-lab6/orders/order"
	"example.com/devops-lab6/orders/repository"
	"example.com/devops-lab6/orders/transport"
	"github.com/go-kit/log"
)

func main() {
	addr := flag.String("http.addr", ":8080", "HTTP listen address")
	flag.Parse()
	logger := log.With(log.NewSyncLogger(log.NewLogfmtLogger(os.Stderr)), "service", "orders")
	if err := serve(*addr, logger); err != nil {
		_ = logger.Log("err", err)
		os.Exit(1)
	}
}

func serve(addr string, logger log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo := repository.NewMemoryRepository()
	// The handout uses /orders/1 as a startup check.
	if err := repo.CreateOrder(ctx, order.Order{
		ID: "1", CustomerID: "cust-demo", RestaurantID: "rest-demo",
		Status: "Pending", CreatedOn: time.Now().Unix(),
	}); err != nil {
		return err
	}
	svc := implementation.NewService(repo)
	server := run.HTTPServer(addr, transport.NewHTTP(endpoint.MakeEndpoints(svc), logger))
	done := make(chan error, 1)
	go func() { done <- run.ServeHTTP(server) }()
	_ = logger.Log("transport", "HTTP", "addr", addr)
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), run.ShutdownTimeout)
		defer cancel()
		return run.ShutdownHTTP(shutdown, server)
	}
}
