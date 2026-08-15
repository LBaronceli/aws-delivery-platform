package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LBaronceli/aws-delivery-platform/app/internal/delivery"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	NewHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}

	var body healthResponse
	decodeResponse(t, response, &body)
	if body.Status != "ok" {
		t.Fatalf("expected status value %q, got %q", "ok", body.Status)
	}
}

func TestDeliveryLifecycle(t *testing.T) {
	api := NewHandler()

	createResponse := performRequest(t, api, http.MethodPost, "/deliveries", `{
		"pickup": "Wellington",
		"destination": "Napier"
	}`)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d: %s", http.StatusCreated, createResponse.Code, createResponse.Body.String())
	}

	var created delivery.Delivery
	decodeResponse(t, createResponse, &created)
	if !strings.HasPrefix(created.ID, "delivery-") {
		t.Fatalf("expected generated delivery ID, got %q", created.ID)
	}
	if created.Pickup != "Wellington" || created.Destination != "Napier" {
		t.Fatalf("unexpected created delivery: %#v", created)
	}
	if created.Status != delivery.StatusCreated {
		t.Fatalf("expected created status, got %q", created.Status)
	}
	if location := createResponse.Header().Get("Location"); location != "/deliveries/"+created.ID {
		t.Fatalf("unexpected Location header %q", location)
	}

	listResponse := performRequest(t, api, http.MethodGet, "/deliveries", "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d", http.StatusOK, listResponse.Code)
	}
	var list deliveryListResponse
	decodeResponse(t, listResponse, &list)
	if len(list.Deliveries) != 1 || list.Deliveries[0].ID != created.ID {
		t.Fatalf("unexpected delivery list: %#v", list.Deliveries)
	}

	getResponse := performRequest(t, api, http.MethodGet, "/deliveries/"+created.ID, "")
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d", http.StatusOK, getResponse.Code)
	}
	var found delivery.Delivery
	decodeResponse(t, getResponse, &found)
	if found != created {
		t.Fatalf("expected %#v, got %#v", created, found)
	}

	updateResponse := performRequest(t, api, http.MethodPatch, "/deliveries/"+created.ID, `{
		"destination": "Auckland",
		"status": "scheduled"
	}`)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update status %d, got %d: %s", http.StatusOK, updateResponse.Code, updateResponse.Body.String())
	}
	var updated delivery.Delivery
	decodeResponse(t, updateResponse, &updated)
	if updated.Pickup != "Wellington" || updated.Destination != "Auckland" || updated.Status != delivery.StatusScheduled {
		t.Fatalf("unexpected updated delivery: %#v", updated)
	}
}

func TestListDeliveriesStartsEmpty(t *testing.T) {
	response := performRequest(t, NewHandler(), http.MethodGet, "/deliveries", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body deliveryListResponse
	decodeResponse(t, response, &body)
	if body.Deliveries == nil {
		t.Fatal("expected an empty JSON array, got null")
	}
	if len(body.Deliveries) != 0 {
		t.Fatalf("expected no deliveries, got %d", len(body.Deliveries))
	}
}

func TestCreateDeliveryValidatesInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing destination", body: `{"pickup":"Wellington"}`},
		{name: "empty pickup", body: `{"pickup":" ","destination":"Napier"}`},
		{name: "unknown field", body: `{"pickup":"Wellington","destination":"Napier","owner":"user-1"}`},
		{name: "invalid JSON", body: `{"pickup":`},
		{name: "multiple objects", body: `{"pickup":"Wellington","destination":"Napier"} {}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(t, NewHandler(), http.MethodPost, "/deliveries", test.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
			}
		})
	}
}

func TestUpdateDeliveryValidatesInput(t *testing.T) {
	api := NewHandler()
	created := createDelivery(t, api)

	tests := []struct {
		name string
		body string
	}{
		{name: "no fields", body: `{}`},
		{name: "empty destination", body: `{"destination":" "}`},
		{name: "invalid status", body: `{"status":"lost"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(t, api, http.MethodPatch, "/deliveries/"+created.ID, test.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
			}
		})
	}
}

func TestDeliveryNotFound(t *testing.T) {
	api := NewHandler()

	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		body := ""
		if method == http.MethodPatch {
			body = `{"status":"cancelled"}`
		}
		response := performRequest(t, api, method, "/deliveries/delivery-missing", body)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s: expected status %d, got %d", method, http.StatusNotFound, response.Code)
		}
	}
}

func TestHealthRejectsOtherMethods(t *testing.T) {
	response := performRequest(t, NewHandler(), http.MethodPost, "/health", "")
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, response.Code)
	}
}

func createDelivery(t *testing.T, api http.Handler) delivery.Delivery {
	t.Helper()
	response := performRequest(t, api, http.MethodPost, "/deliveries", `{"pickup":"Wellington","destination":"Napier"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, response.Code)
	}

	var created delivery.Delivery
	decodeResponse(t, response, &created)
	return created
}

func performRequest(t *testing.T, api http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	api.ServeHTTP(response, request)
	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
