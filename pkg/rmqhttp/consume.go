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

	q, err := rmq.Channel.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := rmq.Channel.Consume(
		q.Name,
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

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var payload rmqPayload
			err := json.Unmarshal(d.Body, &payload)
			if err != nil {
				log.Error(err)
				d.Nack(false, false)
				continue
			}
			resp, err := http.Post(payload.Endpoint, payload.ContentType, strings.NewReader(payload.Content))
			if err != nil {
				log.Error(err)
				d.Nack(false, false)
				continue
			}

			body, _ := ioutil.ReadAll(resp.Body)

			log.Debugf("HTTP %d from %s\n%s", resp.StatusCode, payload.Endpoint, body)

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				log.Error(err)
				d.Nack(false, false)
				continue
			}

			d.Ack(false)
		}
	}()

	log.Infof("Waiting for messages from queue. To exit press CTRL+C")
	<-forever
}
