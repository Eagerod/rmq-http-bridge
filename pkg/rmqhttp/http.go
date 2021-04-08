package rmqhttp

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type HttpController struct {
	rmq   *RMQ
	queue *amqp.Queue
}

func NewHttpController() *HttpController {
	httpController := HttpController{NewRMQ(), nil}
	return &httpController
}

func (hc *HttpController) Connect(connectionString, queueName string) error {
	if err := hc.rmq.ConnectRMQ(connectionString); err != nil {
		return err
	}

	queue, err := hc.rmq.PrepareQueue(queueName)
	if err != nil {
		return err
	}

	hc.queue = queue

	return nil
}

func (hc *HttpController) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header()["Content-Type"] = []string{"application/json"}
	w.Write(NewJsonError(http.StatusText(statusCode), message).Json())
}

func (hc *HttpController) HttpHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		hc.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := NewRMQPayload(body)
	if err != nil {
		hc.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Debugf("Publishing to queue: %s; %d byte payload to: %s",
		hc.queue.Name, len(payload.Content), payload.Endpoint)

	// Note: Retries stays in the body.
	// There may eventually be a need to rewrite the body; it can be
	//   omitted if that ever happens.
	headers := amqp.Table{
		retriesHeaderName:    payload.Retries,
		retryDelayHeaderName: payload.Backoff,
	}

	channel, err := hc.rmq.LockChannel()
	if err != nil {
		hc.respondError(w, http.StatusInternalServerError, "Failed to lock channel")
		return
	}
	defer hc.rmq.UnlockChannel(channel)
	err = channel.Publish(
		"",
		hc.queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers:     headers,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (hc *HttpController) HealthHandler(w http.ResponseWriter, r *http.Request) {
	channel, err := hc.rmq.LockChannel()
	if err != nil {
		hc.respondError(w, http.StatusInternalServerError, "Failed to lock channel")
		return
	}
	defer hc.rmq.UnlockChannel(channel)

	queue := DeadLetterQueueName(hc.queue.Name)

	queueInspect, err := channel.QueueInspect(queue)
	if err != nil {
		hc.respondError(w, http.StatusInternalServerError, "Failed to inspect DLQ")
		return
	}

	if queueInspect.Messages != 0 {
		msg := fmt.Sprintf("DLQ has %d items", queueInspect.Messages)
		hc.respondError(w, http.StatusInternalServerError, msg)
		return
	}
}
