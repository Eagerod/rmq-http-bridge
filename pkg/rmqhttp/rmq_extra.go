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
//   https://docs.particular.net/transports/rabbitmq/delayed-delivery
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
	if err := rmq.ConnectRMQ(connectionString); err != nil {
		return err
	}

	// Delete in reverse order, because of how the bindings are chained.
	for i := DelayInfrastructureBitCount - 1; i >= 0; i-- {
		dirl := NewDelayInfrastructureRoutingLayer(i, DelayInfrastructureBitCount)

		rmq.Channel.QueueDelete(dirl.ActiveRouteQueueName, false, false, false)
		rmq.Channel.ExchangeDelete(dirl.ExchangeName, false, false)
	}

	rmq.Channel.ExchangeDelete(DelayInfrastructureDeliveryExchange, false, false)

	return nil
}

func DelayInfrastructure(connectionString string) error {
	if err := rmq.ConnectRMQ(connectionString); err != nil {
		return err
	}

	// Create the delivery exchange first, then build every layer on top.
	err := rmq.Channel.ExchangeDeclare(
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

		err = rmq.Channel.ExchangeDeclare(
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
		_, err = rmq.Channel.QueueDeclare(
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

		err = rmq.Channel.QueueBind(
			dirl.ActiveRouteQueueName,
			dirl.ActiveRoutingKey,
			dirl.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return err
		}

		err = rmq.Channel.ExchangeBind(
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

	fmt.Fprintf(&outStringBuilder, destQueue)

	return outStringBuilder.String()
}
