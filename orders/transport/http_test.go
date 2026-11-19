package transport_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/devops-lab6/orders/endpoint"
	"example.com/devops-lab6/orders/implementation"
	"example.com/devops-lab6/orders/order"
	"example.com/devops-lab6/orders/repository"
	"example.com/devops-lab6/orders/transport"
	"github.com/go-kit/log"
	"github.com/google/uuid"
)

func TestOrderStatusPersistsAcrossRequests(t *testing.T) {
	service := implementation.NewService(repository.NewMemoryRepository())
	handler := transport.NewHTTP(endpoint.MakeEndpoints(service), log.NewNopLogger())
	request := func(method, path, body string, expected int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		if w.Code != expected {
			t.Fatalf("%s %s: got HTTP %d, want %d: %s", method, path, w.Code, expected, w.Body)
		}
		return w
	}

	created := request(http.MethodPost, "/orders", `{"customer_id":"cust-1","restaurant_id":"rest-42","order_items":[{"product_code":"P001","name":"Pizza","unit_price":550,"quantity":1}]}`, http.StatusCreated)
	var creation endpoint.CreateResponse
	if err := json.Unmarshal(created.Body.Bytes(), &creation); err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(creation.ID); err != nil {
		t.Fatal(err)
	}
	t.Logf("POST /orders -> HTTP %d, id=%s", created.Code, creation.ID)
	path := "/orders/" + creation.ID
	get := func(expected string) {
		t.Helper()
		w := request(http.MethodGet, path, "", http.StatusOK)
		var response struct {
			Order order.Order `json:"order"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		o := response.Order
		if o.ID != creation.ID || o.Status != expected || o.CreatedOn <= 0 ||
			len(o.OrderItems) != 1 || o.OrderItems[0].ProductCode != "P001" {
			t.Fatalf("unexpected order: %+v", o)
		}
		t.Logf("GET /orders/{id} -> HTTP %d, status=%s", w.Code, o.Status)
	}

	get("Pending")
	update := request(http.MethodPatch, path+"/status", `{"status":"Completed"}`, http.StatusOK)
	t.Logf("PATCH /orders/{id}/status -> HTTP %d", update.Code)
	get("Completed")
}
