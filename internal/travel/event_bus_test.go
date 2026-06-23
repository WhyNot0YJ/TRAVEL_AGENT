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
