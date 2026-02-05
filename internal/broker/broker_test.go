package broker

import (
	"github.com/IgorLem99/simple_broker/internal/config"

	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewBroker(t *testing.T) {
	cfg := &config.Config{
		Queues: []config.QueueConfig{
			{Name: "q1", Size: 10, MaxSub: 2},
			{Name: "q2", Size: 5, MaxSub: 1},
		},
	}
	b := New(cfg)
	defer b.Close()

	if len(b.queues) != 2 {
		t.Fatalf("expected 2 queues, got %d", len(b.queues))
	}

	if _, ok := b.queues["q1"]; !ok {
		t.Error("expected queue 'q1' to exist")
	}
	if _, ok := b.queues["q2"]; !ok {
		t.Error("expected queue 'q2' to exist")
	}
}

func TestBroker_GetQueue(t *testing.T) {
	cfg := &config.Config{
		Queues: []config.QueueConfig{
			{Name: "q1", Size: 1, MaxSub: 1},
		},
	}
	b := New(cfg)
	defer b.Close()

	t.Run("existing queue", func(t *testing.T) {
		q, err := b.GetQueue("q1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if q.name != "q1" {
			t.Errorf("expected queue with name 'q1', got '%s'", q.name)
		}
	})

	t.Run("non-existing queue", func(t *testing.T) {
		_, err := b.GetQueue("non-existent")
		if err != ErrQueueNotFound {
			t.Fatalf("expected error %v, got %v", ErrQueueNotFound, err)
		}
	})
}

func TestNewQueue(t *testing.T) {
	qCfg := config.QueueConfig{Name: "test", Size: 10, MaxSub: 5}
	q := NewQueue(qCfg)
	defer q.Close()

	if q.name != "test" {
		t.Errorf("expected name 'test', got '%s'", q.name)
	}
	if q.size != 10 {
		t.Errorf("expected size 10, got %d", q.size)
	}
	if q.maxSub != 5 {
		t.Errorf("expected maxSub 5, got %d", q.maxSub)
	}
	if q.msgs == nil {
		t.Error("msgs slice should be initialized")
	}
	if q.subs == nil {
		t.Error("subs map should be initialized")
	}
}

func TestQueue_Send(t *testing.T) {
	t.Run("send to queue", func(t *testing.T) {
		q := NewQueue(config.QueueConfig{Size: 1})
		defer q.Close()
		err := q.Send("hello")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		q.mu.RLock()
		defer q.mu.RUnlock()
		if len(q.msgs) != 1 || !reflect.DeepEqual(q.msgs[0], "hello") {
			t.Errorf("message was not added to queue correctly")
		}
	})

	t.Run("queue full", func(t *testing.T) {
		q := NewQueue(config.QueueConfig{Size: 1})
		defer q.Close()
		if err := q.Send("world"); err != nil { // Fill the queue
			t.Fatalf("failed to fill queue: %v", err)
		}
		err := q.Send("extra")
		if err != ErrQueueFull {
			t.Fatalf("expected error %v, got %v", ErrQueueFull, err)
		}
	})
}

func TestQueue_Subscribe(t *testing.T) {
	t.Run("subscribe", func(t *testing.T) {
		q := NewQueue(config.QueueConfig{MaxSub: 1})
		defer q.Close()
		sub, err := q.Subscribe()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if sub == nil {
			t.Fatal("subscriber channel should not be nil")
		}
		q.mu.RLock()
		defer q.mu.RUnlock()
		if _, ok := q.subs[sub]; !ok {
			t.Error("subscriber was not added to the map")
		}
	})

	t.Run("too many subscribers", func(t *testing.T) {
		q := NewQueue(config.QueueConfig{MaxSub: 1})
		defer q.Close()
		_, err := q.Subscribe() // First one is ok
		if err != nil {
			t.Fatalf("first subscription failed unexpectedly: %v", err)
		}
		_, err = q.Subscribe() // Second one should fail
		if err != ErrTooManySub {
			t.Fatalf("expected error %v, got %v", ErrTooManySub, err)
		}
	})
}

func TestQueue_Unsubscribe(t *testing.T) {
	q := NewQueue(config.QueueConfig{MaxSub: 1})
	defer q.Close()
	sub, _ := q.Subscribe()

	q.Unsubscribe(sub)

	q.mu.RLock()
	defer q.mu.RUnlock()
	if _, ok := q.subs[sub]; ok {
		t.Error("subscriber should have been removed")
	}

	// Check if channel is closed
	select {
	case _, ok := <-sub:
		if ok {
			t.Error("subscriber channel should be closed")
		}
	default:
		t.Error("channel should be readable (closed)")
	}
}

func TestQueue_Broadcaster(t *testing.T) {
	q := NewQueue(config.QueueConfig{Size: 10, MaxSub: 2})
	defer q.Close()

	sub1, _ := q.Subscribe()
	sub2, _ := q.Subscribe()

	msg := "broadcast test"
	if err := q.Send(msg); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case received := <-sub1:
			if !reflect.DeepEqual(received, msg) {
				t.Errorf("sub1: expected message '%v', got '%v'", msg, received)
			}
		case <-time.After(1 * time.Second):
			t.Error("sub1: timed out waiting for message")
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case received := <-sub2:
			if !reflect.DeepEqual(received, msg) {
				t.Errorf("sub2: expected message '%v', got '%v'", msg, received)
			}
		case <-time.After(1 * time.Second):
			t.Error("sub2: timed out waiting for message")
		}
	}()

	wg.Wait()

	// Check if message is removed from queue
	time.Sleep(100 * time.Millisecond) // give broadcaster time to remove message
	q.mu.RLock()
	defer q.mu.RUnlock()
	if len(q.msgs) != 0 {
		t.Errorf("expected message to be removed from queue, but length is %d", len(q.msgs))
	}
}
