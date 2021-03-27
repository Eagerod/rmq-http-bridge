package rmqhttp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
)

func ConsumeQueue(queueName string) {
	if err := rmq.ConnectRMQ("amqp://guest:guest@rabbitmq:5672/"); err != nil {
		log.Fatal(err)
	}

	queue, err := rmq.PrepareQueue(queueName)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := rmq.Channel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	forever := make(chan bool)

	// Presently only allows for one retry.
	// Will have to build a re-queuing mechanism for proper retry count.
	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var payload rmqPayload
			err := json.Unmarshal(d.Body, &payload)
			if err != nil {
				log.Error(err)
				d.Nack(false, !d.Redelivered)
				continue
			}
			resp, err := http.Post(payload.Endpoint, payload.ContentType, strings.NewReader(payload.Content))
			if err != nil {
				log.Error(err)
				d.Nack(false, !d.Redelivered)
				continue
			}

			body, _ := ioutil.ReadAll(resp.Body)

			log.Debugf("HTTP %d from %s\n%s", resp.StatusCode, payload.Endpoint, body)

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				log.Error(err)
				d.Nack(false, !d.Redelivered)
				continue
			}

			d.Ack(false)
		}
	}()

	log.Infof("Waiting for messages from queue. To exit press CTRL+C")
	<-forever
}
