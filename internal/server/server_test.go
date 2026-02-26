package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	dapr "github.com/dapr/go-sdk/client"
)

type mockClient struct {
	getStateFunc    func(ctx context.Context, store, key string, meta map[string]string) (*dapr.StateItem, error)
	saveStateFunc   func(ctx context.Context, store, key string, data []byte, meta map[string]string, opts ...dapr.StateOption) error
	deleteStateFunc func(ctx context.Context, store, key string, meta map[string]string) error
}

func (m *mockClient) GetState(ctx context.Context, store, key string, meta map[string]string) (*dapr.StateItem, error) {
	if m.getStateFunc != nil {
		return m.getStateFunc(ctx, store, key, meta)
	}
	return nil, nil
}

func (m *mockClient) SaveState(ctx context.Context, store, key string, data []byte, meta map[string]string, opts ...dapr.StateOption) error {
	if m.saveStateFunc != nil {
		return m.saveStateFunc(ctx, store, key, data, meta, opts...)
	}
	return nil
}

func (m *mockClient) DeleteState(ctx context.Context, store, key string, meta map[string]string) error {
	if m.deleteStateFunc != nil {
		return m.deleteStateFunc(ctx, store, key, meta)
	}
	return nil
}

func TestServer_health(t *testing.T) {
	srv := New(&mockClient{}, "store", nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health: got status %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "OK" {
		t.Errorf("health: got body %q, want OK", body)
	}
}

func TestServer_getState_notFound(t *testing.T) {
	mock := &mockClient{
		getStateFunc: func(context.Context, string, string, map[string]string) (*dapr.StateItem, error) {
			return &dapr.StateItem{Key: "k", Value: nil}, nil
		},
	}
	srv := New(mock, "store", nil)
	req := httptest.NewRequest(http.MethodGet, "/state/k", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("getState not found: got status %d, want 404", rec.Code)
	}
}

func TestServer_getState_ok(t *testing.T) {
	mock := &mockClient{
		getStateFunc: func(context.Context, string, string, map[string]string) (*dapr.StateItem, error) {
			return &dapr.StateItem{Key: "k", Value: []byte("v")}, nil
		},
	}
	srv := New(mock, "store", nil)
	req := httptest.NewRequest(http.MethodGet, "/state/k", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("getState ok: got status %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "v" {
		t.Errorf("getState ok: got body %q, want v", body)
	}
}

func TestServer_saveState(t *testing.T) {
	var saved []byte
	mock := &mockClient{
		saveStateFunc: func(_ context.Context, _, _ string, data []byte, _ map[string]string, _ ...dapr.StateOption) error {
			saved = data
			return nil
		},
	}
	srv := New(mock, "store", nil)
	body := bytes.NewReader([]byte("payload"))
	req := httptest.NewRequest(http.MethodPost, "/state/k", body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("saveState: got status %d, want 204", rec.Code)
	}
	if string(saved) != "payload" {
		t.Errorf("saveState: saved %q, want payload", string(saved))
	}
}

func TestServer_deleteState(t *testing.T) {
	called := false
	mock := &mockClient{
		deleteStateFunc: func(context.Context, string, string, map[string]string) error {
			called = true
			return nil
		},
	}
	srv := New(mock, "store", nil)
	req := httptest.NewRequest(http.MethodDelete, "/state/k", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("deleteState: got status %d, want 204", rec.Code)
	}
	if !called {
		t.Error("deleteState: DeleteState was not called")
	}
}
