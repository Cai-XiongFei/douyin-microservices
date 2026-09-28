package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func publishDomainEvent(ctx context.Context, routingKey string, eventID string, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}

	channel, err := NewChannel()
	if err != nil {
		return fmt.Errorf("create RabbitMQ channel failed: %w", err)
	}
	defer channel.Close()

	if err := DeclareEventsExchange(channel); err != nil {
		return fmt.Errorf("declare events exchange failed: %w", err)
	}
	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirm failed: %w", err)
	}
	confirmations := channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	if err := channel.PublishWithContext(
		ctx,
		EventsExchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    eventID,
			Timestamp:    time.Now(),
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish event failed: %w", err)
	}

	select {
	case confirmation, ok := <-confirmations:
		if !ok {
			return fmt.Errorf("publisher confirmation channel closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ rejected event")
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for publisher confirmation failed: %w", ctx.Err())
	}
}

func PublishRelationChanged(ctx context.Context, event RelationChangedEvent) error {
	if event.EventID == "" || event.UserID == 0 || event.ToUserID == 0 {
		return fmt.Errorf("relation event contains invalid identifiers")
	}
	if event.ActionType != 1 && event.ActionType != 2 {
		return fmt.Errorf("action_type must be 1 or 2")
	}
	return publishDomainEvent(ctx, RelationChangedRoutingKey, event.EventID, event)
}

func PublishCommentChanged(ctx context.Context, event CommentChangedEvent) error {
	if event.EventID == "" || event.UserID == 0 || event.VideoID == 0 || event.CommentID == 0 {
		return fmt.Errorf("comment event contains invalid identifiers")
	}
	if event.ActionType != 1 && event.ActionType != 2 {
		return fmt.Errorf("action_type must be 1 or 2")
	}
	return publishDomainEvent(ctx, CommentChangedRoutingKey, event.EventID, event)
}

func PublishMessageCreated(ctx context.Context, event MessageCreatedEvent) error {
	if event.EventID == "" || event.MessageID == 0 || event.FromUserID == 0 || event.ToUserID == 0 {
		return fmt.Errorf("message event contains invalid identifiers")
	}
	if event.Content == "" {
		return fmt.Errorf("message content cannot be empty")
	}
	return publishDomainEvent(ctx, MessageCreatedRoutingKey, event.EventID, event)
}

func PublishVideoPublished(ctx context.Context, event VideoPublishedEvent) error {
	if event.EventID == "" || event.VideoID == 0 || event.AuthorID == 0 {
		return fmt.Errorf("video event contains invalid identifiers")
	}
	return publishDomainEvent(ctx, VideoPublishedRoutingKey, event.EventID, event)
}
