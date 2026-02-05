package handler

import (
	"github.com/IgorLem99/simple_broker/internal/broker"
	"github.com/IgorLem99/simple_broker/internal/config"

	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandler_PostMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := &config.Config{Queues: []config.QueueConfig{{Name: "q1", Size: 1, MaxSub: 1}}}
		b := broker.New(cfg)
		defer b.Close()
		h := New(b)

		body := strings.NewReader(`{"key": "value"}`)
		req := httptest.NewRequest(http.MethodPost, "/queues/q1/messages", body)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
		}
	})

	t.Run("queue not found", func(t *testing.T) {
		b := broker.New(&config.Config{})
		defer b.Close()
		h := New(b)

		body := strings.NewReader(`{"key": "value"}`)
		req := httptest.NewRequest(http.MethodPost, "/queues/non-existent/messages", body)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("queue full", func(t *testing.T) {
		cfg := &config.Config{Queues: []config.QueueConfig{{Name: "q1", Size: 1, MaxSub: 1}}}
		b := broker.New(cfg)
		defer b.Close()
		h := New(b)

		q, _ := b.GetQueue("q1")
		if err := q.Send("first message"); err != nil {
			t.Fatalf("failed to fill queue: %v", err)
		}

		body := strings.NewReader(`{"key": "value"}`)
		req := httptest.NewRequest(http.MethodPost, "/queues/q1/messages", body)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusServiceUnavailable {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
		}
	})

	t.Run("bad request", func(t *testing.T) {
		cfg := &config.Config{Queues: []config.QueueConfig{{Name: "q1", Size: 1, MaxSub: 1}}}
		b := broker.New(cfg)
		defer b.Close()
		h := New(b)

		body := strings.NewReader(`this is not json`)
		req := httptest.NewRequest(http.MethodPost, "/queues/q1/messages", body)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

func TestHandler_PostSubscription(t *testing.T) {
	t.Run("success and message receive", func(t *testing.T) {
		cfg := &config.Config{Queues: []config.QueueConfig{{Name: "q1", Size: 1, MaxSub: 1}}}
		b := broker.New(cfg)
		defer b.Close()
		h := New(b)

		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest(http.MethodPost, "/queues/q1/subscriptions", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			defer close(done)
			h.ServeHTTP(rr, req)
		}()

		time.Sleep(200 * time.Millisecond)

		q, _ := b.GetQueue("q1")
		q.Send("test message")

		time.Sleep(200 * time.Millisecond)
		cancel()
		<-done

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expected := "\"test message\"\n"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %q want %q", rr.Body.String(), expected)
		}
	})

	t.Run("queue not found", func(t *testing.T) {
		b := broker.New(&config.Config{})
		defer b.Close()
		h := New(b)

		req := httptest.NewRequest(http.MethodPost, "/queues/non-existent/subscriptions", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("too many subscribers", func(t *testing.T) {
		cfg := &config.Config{Queues: []config.QueueConfig{{Name: "q1", Size: 1, MaxSub: 1}}}
		b := broker.New(cfg)
		defer b.Close()
		h := New(b)

		q, _ := b.GetQueue("q1")
		_, _ = q.Subscribe()

		req := httptest.NewRequest(http.MethodPost, "/queues/q1/subscriptions", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusServiceUnavailable {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
		}
	})
}

func TestHandler_NotFound(t *testing.T) {
	h := New(nil) // No broker needed for this test

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/queues/q1/messages", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/invalid/path", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}
