package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestConsumeEventsRetriesThenMovesMessageToDLQ(t *testing.T) {
	oldMaxRetries := consumerMaxRetries
	oldRetryDelay := consumerRetryDelay
	consumerMaxRetries = 2
	consumerRetryDelay = 50 * time.Millisecond
	defer func() {
		consumerMaxRetries = oldMaxRetries
		consumerRetryDelay = oldRetryDelay
	}()

	queueName := fmt.Sprintf("test.retry.%d", time.Now().UnixNano())
	routingKey := queueName + ".event"

	setupChannel, err := NewChannel()
	if err != nil {
		t.Fatalf("create setup channel failed: %v", err)
	}
	defer setupChannel.Close()
	if _, err := declareConsumerTopology(setupChannel, queueName, routingKey); err != nil {
		t.Fatalf("declare test topology failed: %v", err)
	}
	defer func() {
		_, _ = setupChannel.QueueDelete(queueName, false, false, false)
		_, _ = setupChannel.QueueDelete(RetryQueueName(queueName), false, false, false)
		_, _ = setupChannel.QueueDelete(DeadLetterQueueName(queueName), false, false, false)
	}()

	deadLetters, err := setupChannel.Consume(
		DeadLetterQueueName(queueName), "", true, true, false, false, nil,
	)
	if err != nil {
		t.Fatalf("consume test dead-letter queue failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumerDone := make(chan error, 1)
	var attempts int32
	go func() {
		consumerDone <- ConsumeEvents(
			ctx,
			queueName,
			routingKey,
			func(context.Context, []byte) (bool, error) {
				atomic.AddInt32(&attempts, 1)
				return true, errors.New("temporary test failure")
			},
		)
	}()
	defer func() {
		cancel()
		select {
		case err := <-consumerDone:
			if err != nil {
				t.Errorf("consumer stopped with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("consumer did not stop after cancellation")
		}
	}()

	// Give ConsumeEvents time to register its consumer before publishing.
	time.Sleep(100 * time.Millisecond)
	if err := setupChannel.PublishWithContext(
		context.Background(),
		EventsExchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    queueName,
			Body:         []byte(`{"test":true}`),
		},
	); err != nil {
		t.Fatalf("publish test message failed: %v", err)
	}

	select {
	case delivery := <-deadLetters:
		if got := retryCount(delivery.Headers); got != consumerMaxRetries {
			t.Fatalf("dead-letter retry count = %d, want %d", got, consumerMaxRetries)
		}
		if _, ok := delivery.Headers[errorHeader]; !ok {
			t.Fatal("dead-letter message has no failure reason")
		}
		if got := atomic.LoadInt32(&attempts); got != int32(consumerMaxRetries+1) {
			t.Fatalf("handler attempts = %d, want %d", got, consumerMaxRetries+1)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for dead-letter message")
	}
}
