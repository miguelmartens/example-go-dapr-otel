package server

import (
	"context"
	"sync"

	dapr "github.com/dapr/go-sdk/client"
)

// MemStore is an in-memory StateClient for local dev without Dapr.
type MemStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemStore creates an in-memory state store.
func NewMemStore() *MemStore {
	return &MemStore{data: make(map[string][]byte)}
}

// GetState returns the value for key, or nil if not found.
func (m *MemStore) GetState(ctx context.Context, _, key string, _ map[string]string) (*dapr.StateItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok || len(v) == 0 {
		return &dapr.StateItem{Key: key, Value: nil}, nil
	}
	return &dapr.StateItem{Key: key, Value: v}, nil
}

// SaveState stores the value for key.
func (m *MemStore) SaveState(ctx context.Context, _, key string, data []byte, _ map[string]string, _ ...dapr.StateOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	return nil
}

// DeleteState removes the key.
func (m *MemStore) DeleteState(ctx context.Context, _, key string, _ map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}
