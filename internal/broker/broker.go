package broker

import (
	"errors"
	"sync"

	"github.com/IgorLem99/simple_broker/internal/config"
)

var (
	ErrQueueFull     = errors.New("queue full")
	ErrTooManySub    = errors.New("too many subscribers")
	ErrQueueNotFound = errors.New("queue not found")
)

type Message any

type Subscriber chan Message

type Queue struct {
	mu     sync.RWMutex
	cond   *sync.Cond
	name   string
	size   int
	maxSub int
	msgs   []Message
	subs   map[Subscriber]struct{}
	done   chan struct{}
	closed chan struct{}
}

func NewQueue(cfg config.QueueConfig) *Queue {
	q := &Queue{
		name:   cfg.Name,
		size:   cfg.Size,
		maxSub: cfg.MaxSub,
		msgs:   make([]Message, 0, cfg.Size),
		subs:   make(map[Subscriber]struct{}),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.mu)
	go q.broadcaster()
	return q
}

func (q *Queue) Subscribe() (Subscriber, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.subs) >= q.maxSub {
		return nil, ErrTooManySub
	}

	sub := make(Subscriber)
	q.subs[sub] = struct{}{}
	q.cond.Signal()

	return sub, nil
}

func (q *Queue) Unsubscribe(sub Subscriber) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.subs[sub]; ok {
		delete(q.subs, sub)
		close(sub)
	}
}

func (q *Queue) Send(msg Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.msgs) >= q.size {
		return ErrQueueFull
	}

	q.msgs = append(q.msgs, msg)
	q.cond.Signal()

	return nil
}

func (q *Queue) broadcaster() {
	defer close(q.closed)

	for {
		q.mu.Lock()
		select {
		case <-q.done:
			q.mu.Unlock()
			return
		default:
		}

		for len(q.msgs) == 0 || len(q.subs) == 0 {
			q.cond.Wait()
			select {
			case <-q.done:
				q.mu.Unlock()
				return
			default:
			}
		}

		msg := q.msgs[0]
		q.msgs = q.msgs[1:]

		subs := make([]Subscriber, 0, len(q.subs))
		for sub := range q.subs {
			subs = append(subs, sub)
		}
		q.mu.Unlock()

		var wg sync.WaitGroup
		for _, sub := range subs {
			wg.Add(1)
			go func(sub Subscriber) {
				defer wg.Done()
				defer func() {
					_ = recover()
				}()
				select {
				case sub <- msg:
				case <-q.done:
				}
			}(sub)
		}

		wg.Wait()
	}
}

func (q *Queue) Close() {
	q.mu.Lock()

	select {
	case <-q.done:
		q.mu.Unlock()
		return
	default:
		close(q.done)
	}
	q.cond.Broadcast()
	q.mu.Unlock()

	<-q.closed

	q.mu.Lock()
	defer q.mu.Unlock()

	for sub := range q.subs {
		delete(q.subs, sub)
		close(sub)
	}
}

type Broker struct {
	mu     sync.RWMutex
	queues map[string]*Queue
}

func New(cfg *config.Config) *Broker {
	b := &Broker{
		queues: make(map[string]*Queue),
	}

	for _, qc := range cfg.Queues {
		b.queues[qc.Name] = NewQueue(qc)
	}

	return b
}

func (b *Broker) GetQueue(name string) (*Queue, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	q, ok := b.queues[name]
	if !ok {
		return nil, ErrQueueNotFound
	}

	return q, nil
}

func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, q := range b.queues {
		q.Close()
	}
}
