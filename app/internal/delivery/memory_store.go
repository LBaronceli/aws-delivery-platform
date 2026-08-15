package delivery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// Store defines the persistence operations used by the HTTP API.
type Store interface {
	Create(ctx context.Context, pickup, destination string) (Delivery, error)
	List(ctx context.Context) ([]Delivery, error)
	Get(ctx context.Context, id string) (Delivery, bool, error)
	Update(ctx context.Context, id string, update Update) (Delivery, bool, error)
}

// MemoryStore keeps deliveries in one process. It is safe for concurrent use.
type MemoryStore struct {
	mu         sync.RWMutex
	deliveries map[string]Delivery
	order      []string
}

// NewMemoryStore returns an empty in-memory delivery store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		deliveries: make(map[string]Delivery),
	}
}

// Create stores a new delivery with the initial created status.
func (store *MemoryStore) Create(_ context.Context, pickup, destination string) (Delivery, error) {
	id, err := newID()
	if err != nil {
		return Delivery{}, err
	}

	created := Delivery{
		ID:          id,
		Pickup:      pickup,
		Destination: destination,
		Status:      StatusCreated,
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	store.deliveries[id] = created
	store.order = append(store.order, id)

	return created, nil
}

// List returns deliveries in creation order.
func (store *MemoryStore) List(_ context.Context) ([]Delivery, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	deliveries := make([]Delivery, 0, len(store.order))
	for _, id := range store.order {
		deliveries = append(deliveries, store.deliveries[id])
	}

	return deliveries, nil
}

// Get returns one delivery by ID.
func (store *MemoryStore) Get(_ context.Context, id string) (Delivery, bool, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	found, ok := store.deliveries[id]
	return found, ok, nil
}

// Update applies the supplied fields to one delivery.
func (store *MemoryStore) Update(_ context.Context, id string, update Update) (Delivery, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	current, ok := store.deliveries[id]
	if !ok {
		return Delivery{}, false, nil
	}

	if update.Pickup != nil {
		current.Pickup = *update.Pickup
	}
	if update.Destination != nil {
		current.Destination = *update.Destination
	}
	if update.Status != nil {
		current.Status = *update.Status
	}

	store.deliveries[id] = current
	return current, true, nil
}

func newID() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate delivery ID: %w", err)
	}

	return "delivery-" + hex.EncodeToString(random[:]), nil
}
