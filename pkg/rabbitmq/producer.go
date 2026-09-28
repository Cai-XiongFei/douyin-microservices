package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareEventsExchange(
	channel *amqp.Channel,
) error {
	return channel.ExchangeDeclare(
		EventsExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
}

func PublishFavoriteChanged(
	ctx context.Context,
	event FavoriteChangedEvent,
) error {
	if event.EventID == "" {
		return fmt.Errorf("event_id cannot be empty")
	}

	if event.UserID == 0 {
		return fmt.Errorf("user_id cannot be zero")
	}

	if event.VideoID == 0 {
		return fmt.Errorf("video_id cannot be zero")
	}

	if event.ActionType != 1 && event.ActionType != 2 {
		return fmt.Errorf("action_type must be 1 or 2")
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal favorite event failed: %w", err)
	}

	channel, err := NewChannel()
	if err != nil {
		return fmt.Errorf("create RabbitMQ channel failed: %w", err)
	}
	defer channel.Close()

	if err := DeclareEventsExchange(channel); err != nil {
		return fmt.Errorf("declare events exchange failed: %w", err)
	}

	// 开启生产者确认，等待 RabbitMQ 确认收到消息。
	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirm failed: %w", err)
	}

	confirmations := channel.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	err = channel.PublishWithContext(
		ctx,
		EventsExchange,
		FavoriteChangedRoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.EventID,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish favorite event failed: %w", err)
	}

	select {
	case confirmation, ok := <-confirmations:
		if !ok {
			return fmt.Errorf(
				"publisher confirmation channel closed",
			)
		}

		if !confirmation.Ack {
			return fmt.Errorf(
				"RabbitMQ rejected favorite event",
			)
		}

		return nil

	case <-ctx.Done():
		return fmt.Errorf(
			"wait for publisher confirmation failed: %w",
			ctx.Err(),
		)
	}
}
