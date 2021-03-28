package rmqhttp

import (
	"encoding/base64"
	"encoding/json"
	"io"
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
			var payload rmqPayload
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				// This is unrecoverable; don't even obey the retry count.
				// Ship this straight to the DLQ.
				log.Error(err)
				d.Nack(false, false)
				continue
			}

			var httpBodyReader io.Reader = strings.NewReader(payload.Content)
			if payload.Base64Decode {
				httpBodyReader = base64.NewDecoder(base64.StdEncoding, httpBodyReader)
			}

			resp, err := http.Post(payload.Endpoint, payload.ContentType, httpBodyReader)
			if err != nil {
				log.Warn(err)
				RequeueOrNack(&rmq, queue, &d)
				continue
			}

			body, _ := ioutil.ReadAll(resp.Body)

			log.Debugf("HTTP %d from %s\n  %s", resp.StatusCode, payload.Endpoint, body)

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				RequeueOrNack(&rmq, queue, &d)
				continue
			}

			d.Ack(false)
		}
	}()

	log.Infof("Waiting for messages from queue. To exit press CTRL+C")
	<-forever
}
