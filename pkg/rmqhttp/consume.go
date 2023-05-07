package rmqhttp

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
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

	client := &http.Client{
		Timeout: time.Second * time.Duration(payload.Timeout),
	}
	req, _ := http.NewRequest("POST", payload.Endpoint, nil)
	req.Body = io.NopCloser(httpBodyReader)

	for key, value := range payload.Headers {
		req.Header.Add(key, value)
	}

	requestStartTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		requestDuration := time.Since(requestStartTime)
		log.Debugf("HTTP fail in %05dms from %s\n  %s", requestDuration.Milliseconds(), payload.Endpoint, err.Error())
		rmq.RequeueOrNack(queue, &delivery)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	requestDuration := time.Since(requestStartTime)
	if len(body) == 0 {
		log.Debugf("HTTP %d in %05dms from %s", resp.StatusCode, requestDuration.Milliseconds(), payload.Endpoint)
	} else {
		log.Debugf("HTTP %d in %05dms from %s\n  %s", resp.StatusCode, requestDuration.Milliseconds(), payload.Endpoint, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		rmq.RequeueOrNack(queue, &delivery)
		return
	}

	delivery.Ack(false)
}

func ConsumeQueue(connectionString, queueName string, consumers int) {
	rmq := NewRMQ()
	if err := rmq.ConnectRMQ(connectionString); err != nil {
		log.Fatal(err)
	}

	queue, err := rmq.PrepareQueue(queueName)
	if err != nil {
		log.Fatal(err)
	}

	wg := sync.WaitGroup{}
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

		wg.Add(1)
		go func() {
			for delivery := range msgs {
				ConsumeOne(rmq, delivery, queue)
			}

			log.Errorf("Channel loop closed.")
			wg.Done()
		}()
	}

	log.Infof("Waiting for messages from queue. To exit press CTRL+C")
	wg.Wait()
}
