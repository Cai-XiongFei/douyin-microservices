package rabbitmq

import (
	"fmt"
	"log"
	"sync"
	"time"

	appViper "douyin/pkg/viper"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	connection      *amqp.Connection
	connectionMu    sync.Mutex
	rabbitMQAddress string
)

var (
	consumerMaxRetries = 3
	consumerRetryDelay = 3 * time.Second
)

func init() {
	config := appViper.Init("rabbitmq")

	host := config.Viper.GetString("rabbitmq.host")
	port := config.Viper.GetInt("rabbitmq.port")
	username := config.Viper.GetString("rabbitmq.username")
	password := config.Viper.GetString("rabbitmq.password")

	if configuredMaxRetries := config.Viper.GetInt("rabbitmq.consumer.max_retries"); configuredMaxRetries > 0 {
		consumerMaxRetries = configuredMaxRetries
	}
	if configuredRetryDelay := config.Viper.GetInt("rabbitmq.consumer.retry_delay_ms"); configuredRetryDelay > 0 {
		consumerRetryDelay = time.Duration(configuredRetryDelay) * time.Millisecond
	}

	rabbitMQAddress = fmt.Sprintf(
		"amqp://%s:%s@%s:%d/",
		username,
		password,
		host,
		port,
	)

	// Best effort only. A broker outage must not stop the business service;
	// NewChannel retries the connection when the relay or a consumer needs it.
	if _, err := ensureConnection(); err != nil {
		log.Printf(
			"RabbitMQ is temporarily unavailable: %v",
			err,
		)
	}
}

func GetConnection() *amqp.Connection {
	currentConnection, err := ensureConnection()
	if err != nil {
		return nil
	}
	return currentConnection
}

func ensureConnection() (*amqp.Connection, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if connection != nil && !connection.IsClosed() {
		return connection, nil
	}
	if rabbitMQAddress == "" {
		return nil, fmt.Errorf("RabbitMQ address is empty")
	}

	newConnection, err := amqp.Dial(rabbitMQAddress)
	if err != nil {
		return nil, fmt.Errorf(
			"connect RabbitMQ failed: %w",
			err,
		)
	}

	connection = newConnection
	log.Printf("RabbitMQ connected successfully")

	return connection, nil
}

// NewChannel creates a channel and reconnects first when necessary.
func NewChannel() (*amqp.Channel, error) {
	currentConnection, err := ensureConnection()
	if err != nil {
		return nil, err
	}

	channel, err := currentConnection.Channel()
	if err == nil {
		return channel, nil
	}

	// The connection can close between ensureConnection and Channel.
	connectionMu.Lock()
	if connection == currentConnection {
		connection = nil
	}
	connectionMu.Unlock()

	currentConnection, reconnectErr := ensureConnection()
	if reconnectErr != nil {
		return nil, reconnectErr
	}

	return currentConnection.Channel()
}
