package rmqhttp

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

import (
	"github.com/streadway/amqp"
)

// Implements this, but without the transport.
// https://docs.particular.net/transports/rabbitmq/delayed-delivery
const DelayInfrastructureBitCount = 28
const DelayInfrastructureDeliveryExchange = "delay-infra-deliver"

type DelayInfrastructureRoutingLayer struct {
	ExchangeName            string
	ActiveRoutingKey        string
	ActiveRouteQueueName    string
	ActiveRouteTtlMs        int64
	InactiveRoutingKey      string
	DestinationExchangeName string
}

func NewDelayInfrastructureRoutingLayer(layer, totalLayers int) DelayInfrastructureRoutingLayer {
	dirl := DelayInfrastructureRoutingLayer{
		ExchangeName:         fmt.Sprintf("delay-infra-%02d", layer),
		ActiveRoutingKey:     HitRoutingKey(layer, totalLayers),
		ActiveRouteQueueName: fmt.Sprintf("delay-queue-%02d", layer),
		ActiveRouteTtlMs:     int64(math.Pow(2, float64(layer))) * 1000,
		InactiveRoutingKey:   MissRoutingKey(layer, totalLayers),
	}

	if layer == 0 {
		dirl.DestinationExchangeName = DelayInfrastructureDeliveryExchange
	} else {
		dirl.DestinationExchangeName = fmt.Sprintf("delay-infra-%02d", layer-1)
	}

	return dirl
}

func HitRoutingKey(index, bitCount int) string {
	nWildcard := bitCount - index - 1
	key := "1.#"
	for i := 0; i < nWildcard; i++ {
		key = "*." + key
	}

	return key
}

func MissRoutingKey(index, bitCount int) string {
	nWildcard := bitCount - index - 1
	key := "0.#"
	for i := 0; i < nWildcard; i++ {
		key = "*." + key
	}

	return key
}

func DestroyInfrastructure(connectionString string) error {
	rmq := NewRMQ()
	if err := rmq.ConnectRMQ(connectionString); err != nil {
		return err
	}

	channel, err := rmq.LockChannel()
	if err != nil {
		return err
	}
	defer rmq.UnlockChannel(channel)

	// Delete in reverse order, because of how the bindings are chained.
	for i := DelayInfrastructureBitCount - 1; i >= 0; i-- {
		dirl := NewDelayInfrastructureRoutingLayer(i, DelayInfrastructureBitCount)

		channel.QueueDelete(dirl.ActiveRouteQueueName, false, false, false)
		channel.ExchangeDelete(dirl.ExchangeName, false, false)
	}

	channel.ExchangeDelete(DelayInfrastructureDeliveryExchange, false, false)

	return nil
}

func DelayInfrastructure(connectionString string) error {
	rmq := NewRMQ()
	if err := rmq.ConnectRMQ(connectionString); err != nil {
		return err
	}

	channel, err := rmq.LockChannel()
	if err != nil {
		return err
	}
	defer rmq.UnlockChannel(channel)

	// Create the delivery exchange first, then build every layer on top.
	err = channel.ExchangeDeclare(
		DelayInfrastructureDeliveryExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for i := 0; i < DelayInfrastructureBitCount; i++ {
		dirl := NewDelayInfrastructureRoutingLayer(i, DelayInfrastructureBitCount)

		fmt.Printf("%+v\n", dirl)

		err = channel.ExchangeDeclare(
			dirl.ExchangeName,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)

		if err != nil {
			return err
		}

		args := amqp.Table{
			"x-dead-letter-exchange": dirl.DestinationExchangeName,
			"x-message-ttl":          dirl.ActiveRouteTtlMs,
		}
		_, err = channel.QueueDeclare(
			dirl.ActiveRouteQueueName,
			true,
			false,
			false,
			false,
			args,
		)
		if err != nil {
			return err
		}

		err = channel.QueueBind(
			dirl.ActiveRouteQueueName,
			dirl.ActiveRoutingKey,
			dirl.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return err
		}

		err = channel.ExchangeBind(
			dirl.DestinationExchangeName,
			dirl.InactiveRoutingKey,
			dirl.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func DelayRoutingExchange() string {
	return fmt.Sprintf("delay-infra-%02d", DelayInfrastructureBitCount-1)
}

func DelayRoutingKey(destQueue string, delaySeconds int64) string {
	binaryRep := strconv.FormatInt(int64(delaySeconds), 2)

	outStringBuilder := strings.Builder{}
	outStringBuilder.Grow(DelayInfrastructureBitCount*2 + 1 + len(destQueue))

	for i := 0; i < DelayInfrastructureBitCount-len(binaryRep); i++ {
		fmt.Fprint(&outStringBuilder, "0.")
	}

	for i := 0; i < len(binaryRep); i++ {
		fmt.Fprintf(&outStringBuilder, "%s.", string(binaryRep[i]))
	}

	fmt.Fprint(&outStringBuilder, destQueue)

	return outStringBuilder.String()
}
