package travel

import "testing"

func TestEventBusSubscribePublishAndCleanup(t *testing.T) {
	bus := NewEventBus()
	events, unsubscribe := bus.Subscribe("task_1")
	if count := bus.SubscriberCount("task_1"); count != 1 {
		t.Fatalf("expected one subscriber, got %d", count)
	}
	bus.Publish(TaskEvent{Type: EventProgress, TaskID: "task_1", Message: "started"})
	event := <-events
	if event.Type != EventProgress || event.Message != "started" {
		t.Fatalf("unexpected event: %#v", event)
	}
	unsubscribe()
	if count := bus.SubscriberCount("task_1"); count != 0 {
		t.Fatalf("expected no subscribers, got %d", count)
	}
}

func TestEventBusReplaysHistoryWithSequence(t *testing.T) {
	bus := NewEventBus()
	bus.Publish(TaskEvent{Type: EventProgress, TaskID: "task_1", Message: "created"})
	bus.Publish(TaskEvent{Type: EventDayDelta, TaskID: "task_1", Message: "day 1"})

	history := bus.History("task_1")
	if len(history) != 2 {
		t.Fatalf("expected two history events, got %#v", history)
	}
	if history[0].Sequence != 1 || history[1].Sequence != 2 {
		t.Fatalf("unexpected history sequence: %#v", history)
	}

	events, unsubscribe := bus.Subscribe("task_1")
	defer unsubscribe()
	first := <-events
	second := <-events
	if first.Message != "created" || second.Message != "day 1" {
		t.Fatalf("unexpected replay events: %#v %#v", first, second)
	}
}
