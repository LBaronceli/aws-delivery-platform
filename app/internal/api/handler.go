package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/LBaronceli/aws-delivery-platform/app/internal/delivery"
)

const maxRequestBodyBytes = 1 << 20

type handler struct {
	store delivery.Store
}

type healthResponse struct {
	Status string `json:"status"`
}

type createDeliveryRequest struct {
	Pickup      string `json:"pickup"`
	Destination string `json:"destination"`
}

type updateDeliveryRequest struct {
	Pickup      *string          `json:"pickup"`
	Destination *string          `json:"destination"`
	Status      *delivery.Status `json:"status"`
}

type deliveryListResponse struct {
	Deliveries []delivery.Delivery `json:"deliveries"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// NewHandler builds the API routes with a new in-memory delivery store.
func NewHandler() http.Handler {
	return NewHandlerWithStore(delivery.NewMemoryStore())
}

// NewHandlerWithStore builds the API routes with the supplied delivery store.
func NewHandlerWithStore(store delivery.Store) http.Handler {
	h := handler{store: store}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /deliveries", h.createDelivery)
	mux.HandleFunc("GET /deliveries", h.listDeliveries)
	mux.HandleFunc("GET /deliveries/{id}", h.getDelivery)
	mux.HandleFunc("PATCH /deliveries/{id}", h.updateDelivery)

	return mux
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (h handler) createDelivery(w http.ResponseWriter, r *http.Request) {
	var request createDeliveryRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	request.Pickup = strings.TrimSpace(request.Pickup)
	request.Destination = strings.TrimSpace(request.Destination)
	if request.Pickup == "" || request.Destination == "" {
		writeError(w, http.StatusBadRequest, "pickup and destination are required")
		return
	}

	created, err := h.store.Create(r.Context(), request.Pickup, request.Destination)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create delivery")
		return
	}

	w.Header().Set("Location", "/deliveries/"+created.ID)
	writeJSON(w, http.StatusCreated, created)
}

func (h handler) listDeliveries(w http.ResponseWriter, r *http.Request) {
	deliveries, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list deliveries")
		return
	}

	writeJSON(w, http.StatusOK, deliveryListResponse{Deliveries: deliveries})
}

func (h handler) getDelivery(w http.ResponseWriter, r *http.Request) {
	found, ok, err := h.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not retrieve delivery")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}

	writeJSON(w, http.StatusOK, found)
}

func (h handler) updateDelivery(w http.ResponseWriter, r *http.Request) {
	var request updateDeliveryRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if request.Pickup == nil && request.Destination == nil && request.Status == nil {
		writeError(w, http.StatusBadRequest, "at least one field is required")
		return
	}
	if request.Pickup != nil {
		trimmed := strings.TrimSpace(*request.Pickup)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "pickup cannot be empty")
			return
		}
		request.Pickup = &trimmed
	}
	if request.Destination != nil {
		trimmed := strings.TrimSpace(*request.Destination)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "destination cannot be empty")
			return
		}
		request.Destination = &trimmed
	}
	if request.Status != nil && !delivery.IsValidStatus(*request.Status) {
		writeError(w, http.StatusBadRequest, "invalid delivery status")
		return
	}

	updated, ok, err := h.store.Update(r.Context(), r.PathValue("id"), delivery.Update{
		Pickup:      request.Pickup,
		Destination: request.Destination,
		Status:      request.Status,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update delivery")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}

	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
