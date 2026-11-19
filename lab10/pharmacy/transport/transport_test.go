package transport_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	pb "example.com/devops-lab6/api/pharmacy/v1"
	"example.com/devops-lab6/pharmacy/endpoint"
	"example.com/devops-lab6/pharmacy/implementation"
	"example.com/devops-lab6/pharmacy/repository"
	"example.com/devops-lab6/pharmacy/transport"
	"github.com/go-kit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestHTTPAndGRPCShareStock(t *testing.T) {
	service := implementation.NewService(repository.NewMemoryRepository())
	endpoints := endpoint.MakeEndpoints(service)
	handler := transport.NewHTTP(endpoints, log.NewNopLogger())
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	checkHTTP := func(w *httptest.ResponseRecorder, want int) {
		t.Helper()
		if w.Code != want {
			t.Fatalf("got HTTP %d, want %d: %s", w.Code, want, w.Body)
		}
	}
	getHTTP := func(id string, want int) {
		t.Helper()
		w := request(http.MethodGet, "/medicines/"+id, "")
		checkHTTP(w, http.StatusOK)
		var response endpoint.GetResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Medicine.ID != id || response.Medicine.Quantity != want {
			t.Fatalf("unexpected medicine: %+v", response.Medicine)
		}
		t.Logf("GET /medicines/{id} -> HTTP %d, quantity=%d", w.Code, response.Medicine.Quantity)
	}

	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pb.RegisterPharmacyServiceServer(server, transport.NewGRPC(endpoints))
	t.Cleanup(server.Stop)
	go func() { _ = server.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := pb.NewPharmacyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	creation := request(http.MethodPost, "/medicines", `{"name":"Парацетамол","manufacturer":"Demo","price":149.5,"prescription":false,"quantity":20}`)
	checkHTTP(creation, http.StatusCreated)
	var created endpoint.CreateResponse
	if err := json.Unmarshal(creation.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created.ID
	t.Logf("POST /medicines -> HTTP %d, id=%s", creation.Code, id)
	getHTTP(id, 20)
	sale := request(http.MethodPost, "/medicines/"+id+"/sell", `{"count":3}`)
	checkHTTP(sale, http.StatusOK)
	t.Logf("POST /medicines/{id}/sell (count=3) -> HTTP %d", sale.Code)
	getHTTP(id, 17)

	retrieved, err := client.GetByID(ctx, &pb.GetByIDRequest{Id: id})
	if err != nil || retrieved.GetMedicine().GetQuantity() != 17 ||
		retrieved.GetMedicine().GetName() != "Парацетамол" ||
		retrieved.GetMedicine().GetPrice() != 149.5 {
		t.Fatalf("gRPC GetByID: %v, %v", retrieved, err)
	}
	t.Logf("gRPC GetByID -> quantity=%d", retrieved.GetMedicine().GetQuantity())
	if _, err := client.Sell(ctx, &pb.SellRequest{Id: id, Count: 4}); err != nil {
		t.Fatal(err)
	}
	t.Log("gRPC Sell (count=4) -> OK")
	getHTTP(id, 13)
	second, err := client.Create(ctx, &pb.CreateRequest{Medicine: &pb.Medicine{
		Name: "Demo", Manufacturer: "Demo", Prescription: true, Quantity: 8,
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("gRPC Create -> id=%s", second.GetId())
	getHTTP(second.GetId(), 8)

	invalid := request(http.MethodPost, "/medicines/"+id+"/sell", `{"count":0}`)
	checkHTTP(invalid, http.StatusBadRequest)
	missing := request(http.MethodGet, "/medicines/unknown", "")
	checkHTTP(missing, http.StatusNotFound)
	conflict := request(http.MethodPost, "/medicines/"+id+"/sell", `{"count":14}`)
	checkHTTP(conflict, http.StatusConflict)
	t.Logf("HTTP invalid count=%d, missing id=%d, insufficient stock=%d", invalid.Code, missing.Code, conflict.Code)
	_, err = client.Sell(ctx, &pb.SellRequest{Id: id, Count: 14})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("gRPC Sell: got %v, want FailedPrecondition", err)
	}
	t.Logf("gRPC Sell (count=14) -> %s", status.Code(err))
	getHTTP(id, 13)

	const parallel = 12
	codesSeen := make(chan int, parallel)
	var wg sync.WaitGroup
	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(http.MethodPost, "/medicines/"+second.GetId()+"/sell", `{"count":1}`)
			codesSeen <- w.Code
		}()
	}
	wg.Wait()
	close(codesSeen)
	totals := make(map[int]int)
	for code := range codesSeen {
		totals[code]++
	}
	if totals[http.StatusOK] != 8 || totals[http.StatusConflict] != 4 {
		t.Fatalf("parallel sales: unexpected results %v", totals)
	}
	final, err := client.GetByID(ctx, &pb.GetByIDRequest{Id: second.GetId()})
	if err != nil || final.GetMedicine().GetQuantity() != 0 {
		t.Fatalf("final stock: %v, %v", final, err)
	}
	t.Logf("Parallel sales: HTTP 200=%d, HTTP 409=%d, final gRPC quantity=%d",
		totals[http.StatusOK], totals[http.StatusConflict], final.GetMedicine().GetQuantity())
}
