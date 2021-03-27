package rmqhttp

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
