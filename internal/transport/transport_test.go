package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type MockSleeper struct {
	sleeps []time.Duration
}

func (m *MockSleeper) Sleep(d time.Duration) {
	m.sleeps = append(m.sleeps, d)
}

func TestSendHeartbeatSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer test-token" {
		t.Errorf("expected Authorization header with Bearer test-token")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		APIURLs:      []string{server.URL},
		TokenCurrent: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{
		"uuid":      "test-uuid",
		"client_id": "test-client",
		"site_id":   "test-site",
	}

	ctx := context.Background()
	sleeper := &MockSleeper{}
	err = client.SendHeartbeat(ctx, payload, DefaultRetryConfig, sleeper)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(sleeper.sleeps) != 0 {
		t.Errorf("expected no sleeps on first success, got %d", len(sleeper.sleeps))
	}
}

func TestSendHeartbeatRetryOn5xx(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer server.Close()

	client, err := New(Config{
		APIURLs:      []string{server.URL},
		TokenCurrent: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{"uuid": "test"}

	ctx := context.Background()
	sleeper := &MockSleeper{}
	err = client.SendHeartbeat(ctx, payload, DefaultRetryConfig, sleeper)
	if err != nil {
		t.Errorf("expected success after retries, got %v", err)
	}

	if int(attempts.Load()) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}

	if len(sleeper.sleeps) != 2 {
		t.Errorf("expected 2 sleeps, got %d", len(sleeper.sleeps))
	}

	expectedDelays := []time.Duration{5 * time.Second, 10 * time.Second}
	for i, expected := range expectedDelays {
		if i < len(sleeper.sleeps) && sleeper.sleeps[i] != expected {
			t.Errorf("sleep %d: expected %v, got %v", i, expected, sleeper.sleeps[i])
		}
	}
}

func TestSendHeartbeatNoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		APIURLs:      []string{server.URL},
		TokenCurrent: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{"uuid": "test"}

	ctx := context.Background()
	sleeper := &MockSleeper{}
	err = client.SendHeartbeat(ctx, payload, DefaultRetryConfig, sleeper)
	if err == nil {
		t.Error("expected error on 4xx, got nil")
	}

	if int(attempts.Load()) != 1 {
		t.Errorf("expected 1 attempt on 4xx, got %d", attempts.Load())
	}

	if len(sleeper.sleeps) != 0 {
		t.Errorf("expected no sleeps on 4xx, got %d", len(sleeper.sleeps))
	}
}

func TestTokenFallback(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		auth := r.Header.Get("Authorization")

		if count == 1 && auth == "Bearer current-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid token"}`))
		} else if auth == "Bearer grace-token" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid token"}`))
		}
	}))
	defer server.Close()

	client, err := New(Config{
		APIURLs:      []string{server.URL},
		TokenCurrent: "current-token",
		TokenGrace:   "grace-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{"uuid": "test"}

	ctx := context.Background()
	sleeper := &MockSleeper{}
	err = client.SendHeartbeat(ctx, payload, DefaultRetryConfig, sleeper)
	if err != nil {
		t.Errorf("expected success with grace token, got %v", err)
	}

	if int(attempts.Load()) != 2 {
		t.Errorf("expected 2 attempts (current + grace), got %d", attempts.Load())
	}
}

func TestMaxRetriesExhausted(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		APIURLs:      []string{server.URL},
		TokenCurrent: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{"uuid": "test"}

	ctx := context.Background()
	sleeper := &MockSleeper{}
	err = client.SendHeartbeat(ctx, payload, DefaultRetryConfig, sleeper)
	if err == nil {
		t.Error("expected error after max retries, got nil")
	}

	expectedAttempts := 1 + DefaultRetryConfig.MaxRetries
	if int(attempts.Load()) != expectedAttempts {
		t.Errorf("expected %d attempts, got %d", expectedAttempts, attempts.Load())
	}

	if len(sleeper.sleeps) != DefaultRetryConfig.MaxRetries {
		t.Errorf("expected %d sleeps, got %d", DefaultRetryConfig.MaxRetries, len(sleeper.sleeps))
	}
}

func TestPayloadMarshaling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}

		if payload["uuid"] != "test-uuid" {
			t.Errorf("expected uuid=test-uuid, got %v", payload["uuid"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{
		APIURLs:      []string{server.URL},
		TokenCurrent: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{
		"uuid":      "test-uuid",
		"client_id": "test-client",
	}

	ctx := context.Background()
	err = client.SendHeartbeat(ctx, payload, DefaultRetryConfig, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSendHeartbeatFallbackOnServerError(t *testing.T) {
	var primaryAttempts atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryAttempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer primary.Close()

	var fallbackAttempts atomic.Int32
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackAttempts.Add(1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer fallback.Close()

	client, err := New(Config{
		APIURLs:      []string{primary.URL, fallback.URL},
		TokenCurrent: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{"uuid": "test"}

	ctx := context.Background()
	sleeper := &MockSleeper{}
	retryConfig := RetryConfig{MaxRetries: 0}
	err = client.SendHeartbeat(ctx, payload, retryConfig, sleeper)
	if err != nil {
		t.Errorf("expected success after fallback, got %v", err)
	}

	if int(primaryAttempts.Load()) != 1 {
		t.Errorf("expected primary attempts=1, got %d", primaryAttempts.Load())
	}
	if int(fallbackAttempts.Load()) != 1 {
		t.Errorf("expected fallback attempts=1, got %d", fallbackAttempts.Load())
	}
}

func TestSendHeartbeatNoFallbackOnClientError(t *testing.T) {
	var primaryAttempts atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryAttempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer primary.Close()

	var fallbackAttempts atomic.Int32
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackAttempts.Add(1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer fallback.Close()

	client, err := New(Config{
		APIURLs:      []string{primary.URL, fallback.URL},
		TokenCurrent: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	payload := map[string]interface{}{"uuid": "test"}

	ctx := context.Background()
	retryConfig := RetryConfig{MaxRetries: 0}
	err = client.SendHeartbeat(ctx, payload, retryConfig, nil)
	if err == nil {
		t.Error("expected error on 4xx, got nil")
	}

	if int(primaryAttempts.Load()) != 1 {
		t.Errorf("expected primary attempts=1, got %d", primaryAttempts.Load())
	}
	if int(fallbackAttempts.Load()) != 0 {
		t.Errorf("expected fallback attempts=0, got %d", fallbackAttempts.Load())
	}
}
