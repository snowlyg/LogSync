package sync

import (
	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	// 队列名称
	QueueName string
	// 交换机
	Exchange string
	// Key
	Key string
	// 连接信息
	Mqurl string
}

// NewRabbitMQ 创建结构体实例
func NewRabbitMQ(queueName, mqurl, exchange, key string) (*RabbitMQ, error) {
	rabbitmq := &RabbitMQ{
		QueueName: queueName,
		Exchange:  exchange,
		Key:       key,
		Mqurl:     mqurl,
	}
	var err error
	// 创建rabbitmq连接
	rabbitmq.conn, err = amqp.Dial(rabbitmq.Mqurl)
	if err != nil {
		return nil, err
	}

	rabbitmq.channel, err = rabbitmq.conn.Channel()
	if err != nil {
		return nil, err
	}

	return rabbitmq, nil
}

// Destory 断开channel和connection
func (r *RabbitMQ) Destory() {
	_ = r.channel.Close()
	_ = r.conn.Close()
}

// NewRabbitMQSimple
// 简单模式Step 1.创建简单模式下的RabbitMq实例
func NewRabbitMQSimple(queueName, mqurl string) (*RabbitMQ, error) {
	return NewRabbitMQ(queueName, mqurl, "", "")
}
