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

func HttpHandler(w http.ResponseWriter, r *http.Request) {
	// I don't know; other validation too.
	queueName := r.URL.Path[1:]

	pathComponents := strings.Split(queueName, "/")
	if len(pathComponents) != 1 {
		w.WriteHeader(http.StatusNotFound)
		w.Header()["Content-Type"] = []string{"application/json"}

		w.Write(NewJsonError("NotFound", "Queue name not valid").Json())
		return
	}
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
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

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Fatal(err)
	}
	var payload rmqPayload
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Publishing to queue: %s; %d byte payload of: %s destined for: %s",
		queueName, len(payload.Content), payload.ContentType, payload.Endpoint)

	err = ch.Publish(
		"",
		q.Name,
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
