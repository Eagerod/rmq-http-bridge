package rmqhttp

import (
	"fmt"
)

import (
	"github.com/streadway/amqp"
)

type rmqPayload struct {
	Endpoint    string
	ContentType string
	Content     string
}

type RMQ struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

func (rmq *RMQ) ConnectRMQ(connectionString string) error {
	if rmq.Connection == nil {
		conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err != nil {
			return err
		}

		rmq.Connection = conn
	}

	if rmq.Channel == nil {
		ch, err := rmq.Connection.Channel()
		if err != nil {
			return err
		}

		rmq.Channel = ch
	}

	return nil
}

// Create the queue we need, and make sure it has a dead letter queue set up.
// This could eventually take in specific configurations that workers/server
//   set up.
func (rmq *RMQ) PrepareQueue(queueName string) (*amqp.Queue, error) {
	if rmq.Channel == nil {
		return nil, fmt.Errorf("Cannot validate queue. RMQ not connected.")
	}

	dlxName := fmt.Sprintf("%s-dead-letter-exchange", queueName)
	dlqName := fmt.Sprintf("%s-dead-letter-queue", queueName)

	if err := rmq.Channel.ExchangeDeclare(dlxName, "fanout", true, false, false, false, nil); err != nil {
		return nil, err
	}

	if _, err := rmq.Channel.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return nil, err
	}

	if err := rmq.Channel.QueueBind(dlqName, "", dlxName, false, nil); err != nil {
		return nil, err
	}

	args := amqp.Table{"x-dead-letter-exchange": dlxName}
	queue, err := rmq.Channel.QueueDeclare(queueName, true, false, false, false, args)
	if err != nil {
		return nil, err
	}

	return &queue, nil
}
