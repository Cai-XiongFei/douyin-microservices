package db

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OutboxStatusPending    = "pending"
	OutboxStatusProcessing = "processing"
	OutboxStatusSent       = "sent"
	OutboxStatusFailed     = "failed"
)

// OutboxEvent 是等待发送到消息队列的领域事件。
type OutboxEvent struct {
	// ID 同时也是发送到RabbitMQ中的event_id。
	// 同一个事件重试时，不能重新生成ID。
	ID string `gorm:"type:varchar(128);primaryKey"`

	// AggregateType 表示事件属于什么业务实体。
	// 例如 favorite、comment、relation。
	AggregateType string `gorm:"type:varchar(50);not null"`

	// AggregateID 可以保存video_id、comment_id等业务ID。
	AggregateID int64 `gorm:"not null;index"`

	// EventType 表示具体事件类型。
	// 例如 favorite.created、favorite.deleted。
	EventType string `gorm:"type:varchar(100);not null"`

	// Payload 保存需要发送到RabbitMQ的JSON数据。
	Payload []byte `gorm:"type:json;not null"`

	// Status：
	// pending    等待发送
	// processing 某个投递器正在处理
	// sent       已成功发送
	Status string `gorm:"type:varchar(20);not null;default:pending;index:idx_outbox_pending,priority:1"`

	RetryCount int `gorm:"not null;default:0"`

	// 失败后不立即重试，通过该字段控制退避时间。
	NextRetryAt time.Time `gorm:"not null;index:idx_outbox_pending,priority:2"`

	// 防止投递器处理过程中崩溃后，事件永久卡在processing。
	LockedBy    string     `gorm:"type:varchar(64);not null;default:'';index"`
	LockedUntil *time.Time `gorm:"index"`

	LastError string `gorm:"type:varchar(1000)"`

	CreatedAt time.Time
	UpdatedAt time.Time
	SentAt    *time.Time
}

// ProcessedEvent 记录某个消费者已经处理过哪些事件。
type ProcessedEvent struct {
	// EventID和ConsumerName组成联合主键。
	EventID string `gorm:"type:varchar(128);primaryKey"`

	// 同一个事件可以被不同消费者分别处理一次。
	ConsumerName string `gorm:"type:varchar(100);primaryKey"`

	ProcessedAt time.Time `gorm:"not null"`
}

var ErrNilTransaction = errors.New("database transaction cannot be nil")

// NewOutboxEvent 创建一条新的Outbox事件。
//
// 注意：
// 事件创建之后，即使后续投递失败并进行重试，也不能重新生成ID。
// 消费者会使用这个ID进行幂等判断。
func NewOutboxEvent(
	eventID string,
	aggregateType string,
	aggregateID int64,
	eventType string,
	payload interface{},
) (*OutboxEvent, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if eventID == "" {
		eventID = uuid.NewString()
	}

	now := time.Now()

	return &OutboxEvent{
		ID:            eventID,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payloadData,
		Status:        OutboxStatusPending,
		RetryCount:    0,
		NextRetryAt:   now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// CreateOutboxEvent 使用传入的数据库事务写入Outbox事件。
//
// 必须传入业务操作正在使用的tx，不能直接使用全局DB，
// 否则业务数据和Outbox事件就不在同一个事务中。
func CreateOutboxEvent(
	tx *gorm.DB,
	event *OutboxEvent,
) error {
	if tx == nil {
		return ErrNilTransaction
	}

	if event == nil {
		return errors.New("outbox event cannot be nil")
	}

	return tx.Create(event).Error
}
