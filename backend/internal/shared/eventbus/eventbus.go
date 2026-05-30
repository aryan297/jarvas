package eventbus

import (
	"context"
	"sync"

	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
)

// Event is the base interface all domain events implement.
type Event interface {
	EventName() string
}

// Handler is a function that processes a domain event.
type Handler func(ctx context.Context, event Event)

// Bus is an in-process, synchronous-by-default event bus.
// Handlers run in goroutines to avoid blocking the publisher.
// This design allows future extraction to an external broker (NATS, Kafka)
// by swapping only this package.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func New() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for the given event name.
func (b *Bus) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

// Publish dispatches the event to all registered handlers concurrently.
// Panics in handlers are recovered and logged so one bad handler cannot
// crash the application.
func (b *Bus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.EventName()]
	b.mu.RUnlock()

	for _, h := range handlers {
		go func(handler Handler) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("event handler panicked",
						zap.String("event", event.EventName()),
						zap.Any("panic", r),
					)
				}
			}()
			handler(ctx, event)
		}(h)
	}
}

// PublishSync dispatches the event synchronously. Use in tests or when
// you need all side effects completed before returning.
func (b *Bus) PublishSync(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.EventName()]
	b.mu.RUnlock()

	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("event handler panicked",
						zap.String("event", event.EventName()),
						zap.Any("panic", r),
					)
				}
			}()
			h(ctx, event)
		}()
	}
}
