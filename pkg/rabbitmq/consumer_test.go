package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRetryCount(t *testing.T) {
	tests := []struct {
		name    string
		headers amqp.Table
		want    int
	}{
		{name: "missing", headers: nil, want: 0},
		{name: "int32", headers: amqp.Table{retryCountHeader: int32(2)}, want: 2},
		{name: "int64", headers: amqp.Table{retryCountHeader: int64(3)}, want: 3},
		{name: "string", headers: amqp.Table{retryCountHeader: "4"}, want: 4},
		{name: "invalid", headers: amqp.Table{retryCountHeader: true}, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := retryCount(test.headers); got != test.want {
				t.Fatalf("retryCount() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestConsumerQueueNames(t *testing.T) {
	if got := RetryQueueName("favorite.cache"); got != "favorite.cache.retry" {
		t.Fatalf("RetryQueueName() = %q", got)
	}
	if got := DeadLetterQueueName("favorite.cache"); got != "favorite.cache.dlq" {
		t.Fatalf("DeadLetterQueueName() = %q", got)
	}
}
