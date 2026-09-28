package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"
)

var ErrOutboxLeaseLost = errors.New("outbox event lease was lost")

// ClaimPendingOutboxEvents atomically leases a batch of due events.
// SKIP LOCKED allows multiple relay instances to work concurrently.
func ClaimPendingOutboxEvents(
	ctx context.Context,
	workerID string,
	batchSize int,
	leaseDuration time.Duration,
) ([]OutboxEvent, error) {
	if workerID == "" {
		return nil, errors.New("worker id cannot be empty")
	}
	if batchSize <= 0 {
		return nil, errors.New("batch size must be greater than zero")
	}
	if leaseDuration <= 0 {
		return nil, errors.New("lease duration must be greater than zero")
	}

	now := time.Now()
	lockedUntil := now.Add(leaseDuration)
	var events []OutboxEvent

	err := GetDB().WithContext(ctx).
		Clauses(dbresolver.Write).
		Transaction(func(tx *gorm.DB) error {
			err := tx.
				Clauses(clause.Locking{
					Strength: "UPDATE",
					Options:  "SKIP LOCKED",
				}).
				Where(
					"(status = ? AND next_retry_at <= ?) OR "+
						"(status = ? AND (locked_until IS NULL OR locked_until <= ?))",
					OutboxStatusPending,
					now,
					OutboxStatusProcessing,
					now,
				).
				Order("created_at ASC").
				Limit(batchSize).
				Find(&events).Error
			if err != nil || len(events) == 0 {
				return err
			}

			ids := make([]string, 0, len(events))
			for index := range events {
				ids = append(ids, events[index].ID)
				events[index].Status = OutboxStatusProcessing
				events[index].LockedBy = workerID
				events[index].LockedUntil = &lockedUntil
			}

			return tx.Model(&OutboxEvent{}).
				Where("id IN ?", ids).
				Updates(map[string]interface{}{
					"status":       OutboxStatusProcessing,
					"locked_by":    workerID,
					"locked_until": lockedUntil,
				}).Error
		})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events failed: %w", err)
	}

	return events, nil
}

func MarkOutboxEventSent(
	ctx context.Context,
	eventID string,
	workerID string,
) error {
	now := time.Now()
	result := GetDB().WithContext(ctx).
		Clauses(dbresolver.Write).
		Model(&OutboxEvent{}).
		Where(
			"id = ? AND status = ? AND locked_by = ?",
			eventID,
			OutboxStatusProcessing,
			workerID,
		).
		Updates(map[string]interface{}{
			"status":       OutboxStatusSent,
			"sent_at":      now,
			"locked_by":    "",
			"locked_until": nil,
			"last_error":   "",
		})
	if result.Error != nil {
		return fmt.Errorf(
			"mark outbox event sent failed: %w",
			result.Error,
		)
	}
	if result.RowsAffected != 1 {
		return ErrOutboxLeaseLost
	}

	return nil
}

// RescheduleOutboxEvent releases the lease after a failed publish.
// Terminal events stay in the failed state for diagnosis and manual replay.
func RescheduleOutboxEvent(
	ctx context.Context,
	eventID string,
	workerID string,
	lastError string,
	nextRetryAt time.Time,
	terminal bool,
) error {
	if len(lastError) > 1000 {
		lastError = lastError[:1000]
	}

	status := OutboxStatusPending
	if terminal {
		status = OutboxStatusFailed
	}

	result := GetDB().WithContext(ctx).
		Clauses(dbresolver.Write).
		Model(&OutboxEvent{}).
		Where(
			"id = ? AND status = ? AND locked_by = ?",
			eventID,
			OutboxStatusProcessing,
			workerID,
		).
		Updates(map[string]interface{}{
			"status":        status,
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": nextRetryAt,
			"locked_by":     "",
			"locked_until":  nil,
			"last_error":    lastError,
		})
	if result.Error != nil {
		return fmt.Errorf(
			"reschedule outbox event failed: %w",
			result.Error,
		)
	}
	if result.RowsAffected != 1 {
		return ErrOutboxLeaseLost
	}

	return nil
}

// RetryFailedOutboxEvent manually moves a terminal event back to pending.
func RetryFailedOutboxEvent(
	ctx context.Context,
	eventID string,
) error {
	now := time.Now()
	result := GetDB().WithContext(ctx).
		Clauses(dbresolver.Write).
		Model(&OutboxEvent{}).
		Where(
			"id = ? AND status = ?",
			eventID,
			OutboxStatusFailed,
		).
		Updates(map[string]interface{}{
			"status":        OutboxStatusPending,
			"retry_count":   0,
			"next_retry_at": now,
			"locked_by":     "",
			"locked_until":  nil,
			"last_error":    "",
		})
	if result.Error != nil {
		return fmt.Errorf(
			"retry failed outbox event failed: %w",
			result.Error,
		)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf(
			"failed outbox event not found: %s",
			eventID,
		)
	}

	return nil
}
