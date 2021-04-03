package rmqhttp

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func ConsumeOne(rmq *RMQ, delivery amqp.Delivery, queue *amqp.Queue) {
	payload, err := NewRMQPayload(delivery.Body)
	if err != nil {
		// This is unrecoverable; don't even obey the retry count.
		// Ship this straight to the DLQ.
		log.Error(err)
		delivery.Nack(false, false)
		return
	}

	var httpBodyReader io.Reader = strings.NewReader(payload.Content)
	if payload.Base64Decode {
		httpBodyReader = base64.NewDecoder(base64.StdEncoding, httpBodyReader)
	}

	client := &http.Client{}
	req, _ := http.NewRequest("POST", payload.Endpoint, nil)
	req.Body = ioutil.NopCloser(httpBodyReader)

	for key, value := range payload.Headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Warn(err)
		rmq.RequeueOrNack(queue, &delivery)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if body == nil || len(body) == 0 {
		log.Debugf("HTTP %d from %s", resp.StatusCode, payload.Endpoint)
	} else {
		log.Debugf("HTTP %d from %s\n  %s", resp.StatusCode, payload.Endpoint, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		rmq.RequeueOrNack(queue, &delivery)
		return
	}

	delivery.Ack(false)
}

func ConsumeQueue(connectionString, queueName string, consumers int) {
	forever := make(chan bool)

	rmq := NewRMQ()
	if err := rmq.ConnectRMQ(connectionString); err != nil {
		log.Fatal(err)
	}

	queue, err := rmq.PrepareQueue(queueName)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < consumers; i++ {
		channel, err := rmq.LockChannel()
		if err != nil {
			log.Fatal(err)
		}

		msgs, err := channel.Consume(
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

		log.Infof("Starting consumer for queue %s", queue.Name)

		go func() {
			for delivery := range msgs {
				ConsumeOne(rmq, delivery, queue)
			}
		}()
	}

	log.Infof("Waiting for messages from queue. To exit press CTRL+C")
	<-forever
}
