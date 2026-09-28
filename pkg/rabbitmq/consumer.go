package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventHandler func(context.Context, []byte) (bool, error)

const (
	RetryExchange      = "douyin.events.retry"
	DeadLetterExchange = "douyin.events.dlx"
	retryCountHeader   = "x-retry-count"
	errorHeader        = "x-error-message"
)

func RetryQueueName(queueName string) string {
	return queueName + ".retry"
}

func DeadLetterQueueName(queueName string) string {
	return queueName + ".dlq"
}

// declareConsumerTopology creates a main queue, a delayed retry queue and a
// dead-letter queue for one business event consumer.
func declareConsumerTopology(channel *amqp.Channel, queueName, routingKey string) (amqp.Queue, error) {
	if err := DeclareEventsExchange(channel); err != nil {
		return amqp.Queue{}, fmt.Errorf("declare events exchange failed: %w", err)
	}
	if err := channel.ExchangeDeclare(RetryExchange, "direct", true, false, false, false, nil); err != nil {
		return amqp.Queue{}, fmt.Errorf("declare retry exchange failed: %w", err)
	}
	if err := channel.ExchangeDeclare(DeadLetterExchange, "direct", true, false, false, false, nil); err != nil {
		return amqp.Queue{}, fmt.Errorf("declare dead-letter exchange failed: %w", err)
	}

	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("declare queue failed: %w", err)
	}
	if err := channel.QueueBind(queue.Name, routingKey, EventsExchange, false, nil); err != nil {
		return amqp.Queue{}, fmt.Errorf("bind queue failed: %w", err)
	}

	// Each retry message has its own expiration. When it expires, RabbitMQ sends
	// it back to the business exchange and the original queue.
	retryQueue, err := channel.QueueDeclare(
		RetryQueueName(queueName), true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    EventsExchange,
			"x-dead-letter-routing-key": routingKey,
		},
	)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("declare retry queue failed: %w", err)
	}
	if err := channel.QueueBind(retryQueue.Name, queueName, RetryExchange, false, nil); err != nil {
		return amqp.Queue{}, fmt.Errorf("bind retry queue failed: %w", err)
	}

	deadLetterQueue, err := channel.QueueDeclare(
		DeadLetterQueueName(queueName), true, false, false, false, nil,
	)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("declare dead-letter queue failed: %w", err)
	}
	if err := channel.QueueBind(deadLetterQueue.Name, queueName, DeadLetterExchange, false, nil); err != nil {
		return amqp.Queue{}, fmt.Errorf("bind dead-letter queue failed: %w", err)
	}
	return queue, nil
}

func retryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	value, exists := headers[retryCountHeader]
	if !exists {
		return 0
	}
	switch count := value.(type) {
	case int8:
		return int(count)
	case int16:
		return int(count)
	case int32:
		return int(count)
	case int64:
		return int(count)
	case int:
		return count
	case string:
		parsed, _ := strconv.Atoi(count)
		return parsed
	default:
		return 0
	}
}

func publishingFromDelivery(delivery amqp.Delivery) amqp.Publishing {
	headers := amqp.Table{}
	for key, value := range delivery.Headers {
		headers[key] = value
	}
	return amqp.Publishing{
		Headers: headers, ContentType: delivery.ContentType,
		ContentEncoding: delivery.ContentEncoding, DeliveryMode: amqp.Persistent,
		Priority: delivery.Priority, CorrelationId: delivery.CorrelationId,
		ReplyTo: delivery.ReplyTo, Expiration: delivery.Expiration,
		MessageId: delivery.MessageId, Timestamp: delivery.Timestamp,
		Type: delivery.Type, UserId: delivery.UserId, AppId: delivery.AppId,
		Body: delivery.Body,
	}
}

func publishAndWaitForConfirm(
	ctx context.Context,
	channel *amqp.Channel,
	confirmations <-chan amqp.Confirmation,
	exchange, routingKey string,
	publishing amqp.Publishing,
) error {
	if err := channel.PublishWithContext(ctx, exchange, routingKey, false, false, publishing); err != nil {
		return err
	}
	select {
	case confirmation, ok := <-confirmations:
		if !ok {
			return fmt.Errorf("publisher confirmation channel closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ rejected the message")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func moveToRetryQueue(
	ctx context.Context,
	channel *amqp.Channel,
	confirmations <-chan amqp.Confirmation,
	queueName string,
	delivery amqp.Delivery,
	nextRetry int,
) error {
	publishing := publishingFromDelivery(delivery)
	publishing.Headers[retryCountHeader] = int32(nextRetry)
	publishing.Expiration = strconv.FormatInt(consumerRetryDelay.Milliseconds(), 10)
	return publishAndWaitForConfirm(ctx, channel, confirmations, RetryExchange, queueName, publishing)
}

func moveToDeadLetterQueue(
	ctx context.Context,
	channel *amqp.Channel,
	confirmations <-chan amqp.Confirmation,
	queueName string,
	delivery amqp.Delivery,
	handleErr error,
) error {
	publishing := publishingFromDelivery(delivery)
	if handleErr != nil {
		errorMessage := handleErr.Error()
		if len(errorMessage) > 512 {
			errorMessage = errorMessage[:512]
		}
		publishing.Headers[errorHeader] = errorMessage
	}
	return publishAndWaitForConfirm(ctx, channel, confirmations, DeadLetterExchange, queueName, publishing)
}

func ConsumeEvents(ctx context.Context, queueName, routingKey string, handler EventHandler) error {
	channel, err := NewChannel()
	if err != nil {
		return fmt.Errorf("create RabbitMQ channel failed: %w", err)
	}
	defer channel.Close()

	queue, err := declareConsumerTopology(channel, queueName, routingKey)
	if err != nil {
		return err
	}
	if err := channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("set consumer qos failed: %w", err)
	}
	// Only acknowledge the original delivery after RabbitMQ has confirmed the
	// retry/dead-letter copy. This prevents message loss.
	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf("enable consumer publisher confirm failed: %w", err)
	}
	confirmations := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume events failed: %w", err)
	}
	log.Printf(
		"event consumer started: queue=%s routing_key=%s max_retries=%d retry_delay=%s dlq=%s",
		queue.Name, routingKey, consumerMaxRetries, consumerRetryDelay, DeadLetterQueueName(queue.Name),
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("delivery channel closed: queue=%s", queue.Name)
			}

			retryable, handleErr := handler(ctx, delivery.Body)
			if handleErr == nil {
				if err := delivery.Ack(false); err != nil {
					log.Printf("ack event failed: queue=%s error=%v", queue.Name, err)
				}
				continue
			}

			currentRetry := retryCount(delivery.Headers)
			if retryable && currentRetry < consumerMaxRetries {
				nextRetry := currentRetry + 1
				if err := moveToRetryQueue(ctx, channel, confirmations, queue.Name, delivery, nextRetry); err != nil {
					log.Printf("send event to retry queue failed: queue=%s error=%v", queue.Name, err)
					_ = delivery.Nack(false, true)
					continue
				}
				log.Printf(
					"event will be retried: queue=%s retry=%d/%d delay=%s error=%v",
					queue.Name, nextRetry, consumerMaxRetries, consumerRetryDelay, handleErr,
				)
				_ = delivery.Ack(false)
				continue
			}

			if err := moveToDeadLetterQueue(ctx, channel, confirmations, queue.Name, delivery, handleErr); err != nil {
				log.Printf("send event to dead-letter queue failed: queue=%s error=%v", queue.Name, err)
				_ = delivery.Nack(false, true)
				continue
			}
			log.Printf(
				"event moved to dead-letter queue: queue=%s retries=%d error=%v",
				queue.Name, currentRetry, handleErr,
			)
			_ = delivery.Ack(false)
		}
	}
}
