package travel

import (
	"sync"
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan TaskEvent]struct{}
}

func NewEventBus() *EventBus {
	return &EventBus{subscribers: map[string]map[chan TaskEvent]struct{}{}}
}

func (b *EventBus) Publish(event TaskEvent) {
	if b == nil {
		return
	}
	b.mu.RLock()
	subs := b.subscribers[event.TaskID]
	channels := make([]chan TaskEvent, 0, len(subs))
	for ch := range subs {
		channels = append(channels, ch)
	}
	b.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *EventBus) Subscribe(taskID string) (<-chan TaskEvent, func()) {
	ch := make(chan TaskEvent, 16)
	if b == nil {
		return ch, func() { close(ch) }
	}
	b.mu.Lock()
	if b.subscribers[taskID] == nil {
		b.subscribers[taskID] = map[chan TaskEvent]struct{}{}
	}
	b.subscribers[taskID][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if subs := b.subscribers[taskID]; subs != nil {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(b.subscribers, taskID)
			}
		}
		close(ch)
	}
	return ch, unsubscribe
}

func (b *EventBus) SubscriberCount(taskID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[taskID])
}
