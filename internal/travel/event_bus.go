package travel

import (
	"sync"
)

const eventHistoryLimit = 100

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan TaskEvent]struct{}
	history     map[string][]TaskEvent
	sequences   map[string]int64
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: map[string]map[chan TaskEvent]struct{}{},
		history:     map[string][]TaskEvent{},
		sequences:   map[string]int64{},
	}
}

func (b *EventBus) Publish(event TaskEvent) {
	if b == nil {
		return
	}
	b.mu.Lock()
	if event.TaskID != "" {
		b.sequences[event.TaskID]++
		event.Sequence = b.sequences[event.TaskID]
		events := append(b.history[event.TaskID], event)
		if len(events) > eventHistoryLimit {
			events = events[len(events)-eventHistoryLimit:]
		}
		b.history[event.TaskID] = events
	}
	subs := b.subscribers[event.TaskID]
	channels := make([]chan TaskEvent, 0, len(subs))
	for ch := range subs {
		channels = append(channels, ch)
	}
	b.mu.Unlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *EventBus) Subscribe(taskID string) (<-chan TaskEvent, func()) {
	if b == nil {
		ch := make(chan TaskEvent)
		return ch, func() { close(ch) }
	}
	b.mu.Lock()
	history := append([]TaskEvent(nil), b.history[taskID]...)
	ch := make(chan TaskEvent, len(history)+16)
	for _, event := range history {
		ch <- event
	}
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

func (b *EventBus) History(taskID string) []TaskEvent {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]TaskEvent(nil), b.history[taskID]...)
}

func (b *EventBus) SubscriberCount(taskID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[taskID])
}
