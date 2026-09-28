package outbox

import (
	"context"
	"fmt"
	"log"
	"time"

	"douyin/dal/db"
	apprabbitmq "douyin/pkg/rabbitmq"

	"github.com/google/uuid"
)

type Config struct {
	BatchSize      int
	PollInterval   time.Duration
	LeaseDuration  time.Duration
	PublishTimeout time.Duration
	MaxRetries     int
	BaseRetryDelay time.Duration
	MaxRetryDelay  time.Duration
}

func DefaultConfig() Config {
	return Config{
		BatchSize:      20,
		PollInterval:   500 * time.Millisecond,
		LeaseDuration:  30 * time.Second,
		PublishTimeout: 5 * time.Second,
		MaxRetries:     10,
		BaseRetryDelay: time.Second,
		MaxRetryDelay:  time.Minute,
	}
}

type Store interface {
	Claim(
		ctx context.Context,
		workerID string,
		batchSize int,
		leaseDuration time.Duration,
	) ([]db.OutboxEvent, error)
	MarkSent(
		ctx context.Context,
		eventID string,
		workerID string,
	) error
	Reschedule(
		ctx context.Context,
		eventID string,
		workerID string,
		lastError string,
		nextRetryAt time.Time,
		terminal bool,
	) error
}

type Publisher interface {
	Publish(
		ctx context.Context,
		routingKey string,
		eventID string,
		payload []byte,
	) error
}

type DBStore struct{}

func (DBStore) Claim(
	ctx context.Context,
	workerID string,
	batchSize int,
	leaseDuration time.Duration,
) ([]db.OutboxEvent, error) {
	return db.ClaimPendingOutboxEvents(
		ctx,
		workerID,
		batchSize,
		leaseDuration,
	)
}

func (DBStore) MarkSent(
	ctx context.Context,
	eventID string,
	workerID string,
) error {
	return db.MarkOutboxEventSent(ctx, eventID, workerID)
}

func (DBStore) Reschedule(
	ctx context.Context,
	eventID string,
	workerID string,
	lastError string,
	nextRetryAt time.Time,
	terminal bool,
) error {
	return db.RescheduleOutboxEvent(
		ctx,
		eventID,
		workerID,
		lastError,
		nextRetryAt,
		terminal,
	)
}

type RabbitPublisher struct{}

func (RabbitPublisher) Publish(
	ctx context.Context,
	routingKey string,
	eventID string,
	payload []byte,
) error {
	return apprabbitmq.PublishOutboxEvent(
		ctx,
		routingKey,
		eventID,
		payload,
	)
}

type Relay struct {
	workerID  string
	config    Config
	store     Store
	publisher Publisher
	now       func() time.Time
}

func NewRelay(config Config) (*Relay, error) {
	return newRelay(
		config,
		DBStore{},
		RabbitPublisher{},
	)
}

func newRelay(
	config Config,
	store Store,
	publisher Publisher,
) (*Relay, error) {
	if config.BatchSize <= 0 {
		return nil, fmt.Errorf(
			"outbox batch size must be greater than zero",
		)
	}
	if config.PollInterval <= 0 {
		return nil, fmt.Errorf(
			"outbox poll interval must be greater than zero",
		)
	}
	if config.LeaseDuration <= 0 {
		return nil, fmt.Errorf(
			"outbox lease duration must be greater than zero",
		)
	}
	if config.PublishTimeout <= 0 {
		return nil, fmt.Errorf(
			"outbox publish timeout must be greater than zero",
		)
	}
	if config.MaxRetries <= 0 {
		return nil, fmt.Errorf(
			"outbox max retries must be greater than zero",
		)
	}
	if config.BaseRetryDelay <= 0 {
		return nil, fmt.Errorf(
			"outbox base retry delay must be greater than zero",
		)
	}
	if config.MaxRetryDelay < config.BaseRetryDelay {
		return nil, fmt.Errorf(
			"outbox max retry delay cannot be less than base retry delay",
		)
	}
	if store == nil {
		return nil, fmt.Errorf("outbox store cannot be nil")
	}
	if publisher == nil {
		return nil, fmt.Errorf(
			"outbox publisher cannot be nil",
		)
	}

	return &Relay{
		workerID:  uuid.NewString(),
		config:    config,
		store:     store,
		publisher: publisher,
		now:       time.Now,
	}, nil
}

// Run polls until ctx is canceled. Errors are logged and retried; a temporary
// MySQL or RabbitMQ outage must not stop the relay permanently.
func (relay *Relay) Run(ctx context.Context) {
	log.Printf(
		"outbox relay started: worker_id=%s batch_size=%d",
		relay.workerID,
		relay.config.BatchSize,
	)

	relay.processAndLog(ctx)

	ticker := time.NewTicker(relay.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf(
				"outbox relay stopped: worker_id=%s",
				relay.workerID,
			)
			return
		case <-ticker.C:
			relay.processAndLog(ctx)
		}
	}
}

func (relay *Relay) processAndLog(ctx context.Context) {
	for {
		count, err := relay.ProcessOnce(ctx)
		if err != nil {
			log.Printf(
				"outbox relay iteration failed: worker_id=%s error=%v",
				relay.workerID,
				err,
			)
			return
		}

		if count < relay.config.BatchSize {
			return
		}
	}
}

// ProcessOnce claims and handles at most one batch. It is exported to make
// deterministic unit and integration testing possible.
func (relay *Relay) ProcessOnce(
	ctx context.Context,
) (int, error) {
	events, err := relay.store.Claim(
		ctx,
		relay.workerID,
		relay.config.BatchSize,
		relay.config.LeaseDuration,
	)
	if err != nil {
		return 0, err
	}

	for index := range events {
		event := events[index]
		publishCtx, cancel := context.WithTimeout(
			ctx,
			relay.config.PublishTimeout,
		)
		publishErr := relay.publisher.Publish(
			publishCtx,
			event.EventType,
			event.ID,
			event.Payload,
		)
		cancel()

		if publishErr == nil {
			if err := relay.store.MarkSent(
				ctx,
				event.ID,
				relay.workerID,
			); err != nil {
				log.Printf(
					"mark outbox event sent failed: event_id=%s error=%v",
					event.ID,
					err,
				)
			}
			continue
		}

		attempt := event.RetryCount + 1
		terminal := attempt >= relay.config.MaxRetries
		nextRetryAt := relay.now().Add(
			relay.retryDelay(attempt),
		)

		if err := relay.store.Reschedule(
			ctx,
			event.ID,
			relay.workerID,
			publishErr.Error(),
			nextRetryAt,
			terminal,
		); err != nil {
			log.Printf(
				"reschedule outbox event failed: event_id=%s error=%v",
				event.ID,
				err,
			)
			continue
		}

		if terminal {
			log.Printf(
				"outbox event moved to failed state: event_id=%s attempts=%d error=%v",
				event.ID,
				attempt,
				publishErr,
			)
		} else {
			log.Printf(
				"outbox publish failed and will retry: event_id=%s attempt=%d next_retry_at=%s error=%v",
				event.ID,
				attempt,
				nextRetryAt.Format(time.RFC3339),
				publishErr,
			)
		}
	}

	return len(events), nil
}

func (relay *Relay) retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	delay := relay.config.BaseRetryDelay
	for current := 1; current < attempt; current++ {
		if delay >= relay.config.MaxRetryDelay/2 {
			return relay.config.MaxRetryDelay
		}
		delay *= 2
	}

	if delay > relay.config.MaxRetryDelay {
		return relay.config.MaxRetryDelay
	}
	return delay
}
