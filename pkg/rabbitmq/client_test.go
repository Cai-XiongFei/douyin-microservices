package rabbitmq

import (
	"testing"
)

func TestRabbitMQConnection(t *testing.T) {
	if GetConnection() == nil {
		t.Fatal("RabbitMQ connection is nil")
	}

	if GetConnection().IsClosed() {
		t.Fatal("RabbitMQ connection is closed")
	}

	channel, err := NewChannel()
	if err != nil {
		t.Fatalf("创建 RabbitMQ Channel 失败：%v", err)
	}
	defer channel.Close()

	// 创建一个临时测试队列。
	// 测试结束、Channel 关闭后，该队列会自动删除。
	queue, err := channel.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("创建 RabbitMQ 测试队列失败：%v", err)
	}

	t.Logf(
		"RabbitMQ 连接成功，临时队列：%s",
		queue.Name,
	)
}
