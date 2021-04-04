package rmqhttp

import (
	"fmt"
	"math"
	"sync"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

const retriesHeaderName string = "x-remaining-retries"
const attemptsHeaderName string = "x-attempt-number"

type RMQ struct {
	Connection       *amqp.Connection
	Channels         []*amqp.Channel
	ChannelQueueLock sync.Mutex
	QueueCache       map[string]*amqp.Queue
}

func NewRMQ() *RMQ {
	rmq := RMQ{nil, nil, sync.Mutex{}, make(map[string]*amqp.Queue)}
	return &rmq
}

func (rmq *RMQ) ConnectRMQ(connectionString string) error {
	if rmq.Connection == nil {
		conn, err := amqp.Dial(connectionString)
		if err != nil {
			return err
		}

		rmq.Connection = conn
	}

	return nil
}

func (rmq *RMQ) LockChannel() (*amqp.Channel, error) {
	rmq.ChannelQueueLock.Lock()
	defer rmq.ChannelQueueLock.Unlock()
	if len(rmq.Channels) == 0 {
		return rmq.Connection.Channel()
	}

	channel := rmq.Channels[0]
	rmq.Channels = rmq.Channels[1:]
	return channel, nil
}

func (rmq *RMQ) UnlockChannel(channel *amqp.Channel) {
	rmq.ChannelQueueLock.Lock()
	defer rmq.ChannelQueueLock.Unlock()
	rmq.Channels = append(rmq.Channels, channel)
}

func DeadLetterQueueName(queue string) string {
	return fmt.Sprintf("%s-dead-letter-queue", queue)
}

// Create the queue we need, and make sure it has a dead letter queue set up.
// This could eventually take in specific configurations that workers/server
//   set up.
func (rmq *RMQ) PrepareQueue(queueName string) (*amqp.Queue, error) {
	channel, err := rmq.LockChannel()
	if err != nil {
		return nil, fmt.Errorf("Cannot validate queue. RMQ not connected.")
	}
	defer rmq.UnlockChannel(channel)

	if queue, ok := rmq.QueueCache[queueName]; ok {
		return queue, nil
	}

	dlxName := fmt.Sprintf("%s-dead-letter-exchange", queueName)
	dlqName := DeadLetterQueueName(queueName)
	delayxName := fmt.Sprintf("%s-delay-delivery", queueName)

	if err := channel.ExchangeDeclare(dlxName, "fanout", true, false, false, false, nil); err != nil {
		return nil, err
	}

	if _, err := channel.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return nil, err
	}

	if err := channel.QueueBind(dlqName, "", dlxName, false, nil); err != nil {
		return nil, err
	}

	args := amqp.Table{"x-dead-letter-exchange": dlxName}
	queue, err := channel.QueueDeclare(queueName, true, false, false, false, args)
	if err != nil {
		return nil, err
	}

	// Because of how the delaying infrastructure works, create an exchange
	//   for the queue, and bind the exchange to the delivery exchange.
	if err := channel.ExchangeDeclare(delayxName, "fanout", true, false, false, false, nil); err != nil {
		return nil, err
	}

	routingKey := fmt.Sprintf("#.%s", queueName)
	if err := channel.ExchangeBind(delayxName, routingKey, DelayInfrastructureDeliveryExchange, false, nil); err != nil {
		return nil, err
	}

	if err := channel.QueueBind(queueName, "", delayxName, false, nil); err != nil {
		return nil, err
	}

	rmq.QueueCache[queue.Name] = &queue

	return &queue, nil
}

func (rmq *RMQ) RequeueOrNack(queue *amqp.Queue, delivery *amqp.Delivery) {
	retries, ok := delivery.Headers[retriesHeaderName]
	if !ok {
		// I guess assume that the retries have been exhausted?
		log.Warn("Retries header not found")
		delivery.Nack(false, false)
		return
	}

	attempts, ok := delivery.Headers[attemptsHeaderName]
	if !ok {
		attempts = 0
	}

	retriesInt, err := ToInt(retries)
	if err != nil {
		log.Error(err)
		delivery.Nack(false, false)
		return
	}

	attemptsInt, err := ToInt(attempts)
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
	delivery.Headers[attemptsHeaderName] = attemptsInt + 1

	// Publish this message back to the queue and Ack the one with the current
	//   retry count.
	channel, err := rmq.LockChannel()
	if err != nil {
		log.Info("Failed to lock channel for requeuing delivery.")
		delivery.Nack(false, true)
		return
	}
	defer rmq.UnlockChannel(channel)

	err = channel.Publish(
		DelayRoutingExchange(),
		DelayRoutingKey(queue.Name, int64(math.Pow(2, float64(attemptsInt)))),
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
