package rmqhttp

import (
	"fmt"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Desribes the primary payload of the system.
//
// Endpoint:     URL where the content will be sent.
// ContentType:  Value of HTTP content type header to send.
// Content:      Payload to send in HTTP request.
// Base64Decode: Whether or not the service needs to decode the given content
//               before sending it.
// Retries       Number of retries before pushing to the DLX.
//				 Defaults to 2; maximum 9.
type rmqPayload struct {
	Endpoint     string
	ContentType  string
	Content      string
	Base64Decode bool
	Retries      int
}

const retriesHeaderName string = "x-remaining-retries"

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

func RequeueOrNack(rmq *RMQ, queue *amqp.Queue, delivery *amqp.Delivery) {
	retries, ok := delivery.Headers[retriesHeaderName]
	if !ok {
		// I guess assume that the retries have been exhausted?
		log.Warn("Retries header not found")
		delivery.Nack(false, false)
		return
	}

	retriesInt, err := ToInt(retries)
	if err != nil {
		log.Error(err)
		delivery.Nack(false, false)
		return
	}

	if retriesInt <= 0 {
		log.Info("Message failed final retry. Sending to DLX.")
		delivery.Nack(false, false)
		return
	}

	delivery.Headers[retriesHeaderName] = retriesInt - 1

	// Publish this message back to the queue and Ack the one with the current
	//   retry count.
	err = rmq.Channel.Publish(
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: delivery.ContentType,
			Body:        delivery.Body,
			Headers:     delivery.Headers,
		},
	)
	if err != nil {
		// Nack and requeue I guess? It will end up getting an extra retry,
		//   but better than DLQing it right away?
		log.Warn("Failed message failed to decrement retries")
		delivery.Nack(false, true)
	} else {
		delivery.Ack(false)
	}
}
