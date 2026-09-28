package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishOutboxEvent publishes the exact JSON stored in the outbox row.
// The caller may mark the row as sent only after this function returns nil.
func PublishOutboxEvent(
	ctx context.Context,
	routingKey string,
	eventID string,
	payload []byte,
) error {
	if routingKey == "" {
		return fmt.Errorf("routing key cannot be empty")
	}
	if eventID == "" {
		return fmt.Errorf("event id cannot be empty")
	}
	if !json.Valid(payload) {
		return fmt.Errorf("outbox payload is not valid JSON")
	}

	channel, err := NewChannel()
	if err != nil {
		return fmt.Errorf(
			"create RabbitMQ channel failed: %w",
			err,
		)
	}
	defer channel.Close()

	if err := DeclareEventsExchange(channel); err != nil {
		return fmt.Errorf(
			"declare events exchange failed: %w",
			err,
		)
	}
	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf(
			"enable publisher confirm failed: %w",
			err,
		)
	}

	confirmations := channel.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)
	returns := channel.NotifyReturn(
		make(chan amqp.Return, 1),
	)

	err = channel.PublishWithContext(
		ctx,
		EventsExchange,
		routingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    eventID,
			Timestamp:    time.Now(),
			Body:         payload,
		},
	)
	if err != nil {
		return fmt.Errorf("publish outbox event failed: %w", err)
	}

	for {
		select {
		case returned, ok := <-returns:
			if !ok {
				return fmt.Errorf(
					"RabbitMQ return channel closed",
				)
			}
			return fmt.Errorf(
				"RabbitMQ returned unroutable event: code=%d text=%s routing_key=%s",
				returned.ReplyCode,
				returned.ReplyText,
				returned.RoutingKey,
			)

		case confirmation, ok := <-confirmations:
			if !ok {
				return fmt.Errorf(
					"publisher confirmation channel closed",
				)
			}
			if !confirmation.Ack {
				return fmt.Errorf(
					"RabbitMQ rejected outbox event",
				)
			}

			// RabbitMQ sends basic.return before basic.ack for an unroutable
			// mandatory message. Check an already-delivered return once more.
			select {
			case returned := <-returns:
				return fmt.Errorf(
					"RabbitMQ returned unroutable event: code=%d text=%s routing_key=%s",
					returned.ReplyCode,
					returned.ReplyText,
					returned.RoutingKey,
				)
			default:
				return nil
			}

		case <-ctx.Done():
			return fmt.Errorf(
				"wait for publisher confirmation failed: %w",
				ctx.Err(),
			)
		}
	}
}
