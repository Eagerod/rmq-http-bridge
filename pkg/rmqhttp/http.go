package rmqhttp

import (
	"io/ioutil"
	"net/http"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header()["Content-Type"] = []string{"application/json"}
	w.Write(NewJsonError(http.StatusText(statusCode), message).Json())
}

func HttpHandler(connectionString, queueName string) func(w http.ResponseWriter, r *http.Request) {
	// Will have to re-evaluate if this ever gets more endpoints.
	rmq := NewRMQ()

	if err := rmq.ConnectRMQ(connectionString); err != nil {
		log.Fatal(err)
	}

	queue, err := rmq.PrepareQueue(queueName)
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}

		payload, err := NewRMQPayload(body)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		log.Debugf("Publishing to queue: %s; %d byte payload to: %s",
			queueName, len(payload.Content), payload.Endpoint)

		// Note: Retries stays in the body.
		// There may eventually be a need to rewrite the body; it can be
		//   omitted if that ever happens.
		headers := amqp.Table{retriesHeaderName: payload.Retries}

		channel, err := rmq.LockChannel()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to lock channel")
			return
		}
		defer rmq.UnlockChannel(channel)
		err = channel.Publish(
			"",
			queue.Name,
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
}
