package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"douyin/dal/db"
)

type fakeStore struct {
	events      []db.OutboxEvent
	claimErr    error
	sentIDs     []string
	rescheduled []rescheduledEvent
}

type rescheduledEvent struct {
	eventID     string
	workerID    string
	lastError   string
	nextRetryAt time.Time
	terminal    bool
}

func (store *fakeStore) Claim(
	context.Context,
	string,
	int,
	time.Duration,
) ([]db.OutboxEvent, error) {
	return store.events, store.claimErr
}

func (store *fakeStore) MarkSent(
	_ context.Context,
	eventID string,
	_ string,
) error {
	store.sentIDs = append(store.sentIDs, eventID)
	return nil
}

func (store *fakeStore) Reschedule(
	_ context.Context,
	eventID string,
	workerID string,
	lastError string,
	nextRetryAt time.Time,
	terminal bool,
) error {
	store.rescheduled = append(
		store.rescheduled,
		rescheduledEvent{
			eventID:     eventID,
			workerID:    workerID,
			lastError:   lastError,
			nextRetryAt: nextRetryAt,
			terminal:    terminal,
		},
	)
	return nil
}

type fakePublisher struct {
	err       error
	published []string
}

func (publisher *fakePublisher) Publish(
	_ context.Context,
	_ string,
	eventID string,
	_ []byte,
) error {
	publisher.published = append(
		publisher.published,
		eventID,
	)
	return publisher.err
}

func testConfig() Config {
	return Config{
		BatchSize:      10,
		PollInterval:   time.Second,
		LeaseDuration:  30 * time.Second,
		PublishTimeout: time.Second,
		MaxRetries:     3,
		BaseRetryDelay: time.Second,
		MaxRetryDelay:  8 * time.Second,
	}
}

func TestProcessOnceMarksPublishedEventSent(t *testing.T) {
	store := &fakeStore{
		events: []db.OutboxEvent{
			{
				ID:        "event-1",
				EventType: "favorite.changed",
				Payload:   []byte(`{"event_id":"event-1"}`),
			},
		},
	}
	publisher := &fakePublisher{}

	relay, err := newRelay(testConfig(), store, publisher)
	if err != nil {
		t.Fatalf("create relay failed: %v", err)
	}

	count, err := relay.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process outbox failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one event, got %d", count)
	}
	if len(store.sentIDs) != 1 || store.sentIDs[0] != "event-1" {
		t.Fatalf("event was not marked sent: %#v", store.sentIDs)
	}
	if len(store.rescheduled) != 0 {
		t.Fatalf(
			"successful event should not be rescheduled: %#v",
			store.rescheduled,
		)
	}
}

func TestProcessOnceReschedulesFailedPublish(t *testing.T) {
	store := &fakeStore{
		events: []db.OutboxEvent{
			{
				ID:         "event-2",
				EventType:  "favorite.changed",
				Payload:    []byte(`{"event_id":"event-2"}`),
				RetryCount: 0,
			},
		},
	}
	publisher := &fakePublisher{
		err: errors.New("broker unavailable"),
	}

	relay, err := newRelay(testConfig(), store, publisher)
	if err != nil {
		t.Fatalf("create relay failed: %v", err)
	}

	now := time.Date(
		2026,
		time.September,
		28,
		12,
		0,
		0,
		0,
		time.Local,
	)
	relay.now = func() time.Time {
		return now
	}

	if _, err := relay.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("process outbox failed: %v", err)
	}
	if len(store.rescheduled) != 1 {
		t.Fatalf(
			"expected one rescheduled event, got %d",
			len(store.rescheduled),
		)
	}

	result := store.rescheduled[0]
	if result.terminal {
		t.Fatal("first failure should not be terminal")
	}
	if !result.nextRetryAt.Equal(now.Add(time.Second)) {
		t.Fatalf(
			"unexpected next retry time: %s",
			result.nextRetryAt,
		)
	}
}

func TestProcessOnceMovesExhaustedEventToFailed(t *testing.T) {
	store := &fakeStore{
		events: []db.OutboxEvent{
			{
				ID:         "event-3",
				EventType:  "favorite.changed",
				Payload:    []byte(`{"event_id":"event-3"}`),
				RetryCount: 2,
			},
		},
	}
	publisher := &fakePublisher{
		err: errors.New("broker unavailable"),
	}

	relay, err := newRelay(testConfig(), store, publisher)
	if err != nil {
		t.Fatalf("create relay failed: %v", err)
	}

	if _, err := relay.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("process outbox failed: %v", err)
	}
	if len(store.rescheduled) != 1 {
		t.Fatalf(
			"expected one rescheduled event, got %d",
			len(store.rescheduled),
		)
	}
	if !store.rescheduled[0].terminal {
		t.Fatal("third failure should move event to failed")
	}
}

func TestRetryDelayIsCapped(t *testing.T) {
	relay, err := newRelay(
		testConfig(),
		&fakeStore{},
		&fakePublisher{},
	)
	if err != nil {
		t.Fatalf("create relay failed: %v", err)
	}

	if actual := relay.retryDelay(20); actual != 8*time.Second {
		t.Fatalf(
			"expected capped retry delay, got %s",
			actual,
		)
	}
}
