package rmqhttp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var rmq RMQ

func HttpHandler(w http.ResponseWriter, r *http.Request) {
	// I don't know; other validation too?
	queueName := r.URL.Path[1:]

	// The router should be handling this, so this is kind of silly.
	pathComponents := strings.Split(queueName, "/")
	if len(pathComponents) != 1 {
		w.WriteHeader(http.StatusNotFound)
		w.Header()["Content-Type"] = []string{"application/json"}
		w.Write(NewJsonError("NotFound", "Queue name not valid").Json())
		return
	}

	if err := rmq.ConnectRMQ("amqp://guest:guest@rabbitmq:5672/"); err != nil {
		log.Fatal(err)
	}

	queue, err := rmq.PrepareQueue(queueName)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	var payload rmqPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header()["Content-Type"] = []string{"application/json"}
		w.Write(NewJsonError("BadRequest", "Invalid JSON").Json())
		return
	}

	log.Debugf("Publishing to queue: %s; %d byte payload of: %s destined for: %s",
		queueName, len(payload.Content), payload.ContentType, payload.Endpoint)

	err = rmq.Channel.Publish(
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusNoContent)
}
