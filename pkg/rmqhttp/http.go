package rmqhttp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
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
	chanLock := sync.Mutex{}

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

		payload := rmqPayload{Retries: 2}
		if err := json.Unmarshal(body, &payload); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		if payload.Endpoint == "" {
			respondError(w, http.StatusBadRequest, "No endpoint given")
			return
		}

		if payload.Retries < 0 || payload.Retries > 9 {
			respondError(w, http.StatusBadRequest, "Retries not within (0, 9)")
			return
		}

		log.Debugf("Publishing to queue: %s; %d byte payload to: %s",
			queueName, len(payload.Content), payload.Endpoint)

		// Note: Retries stays in the body.
		// There may eventually be a need to rewrite the body; it can be
		//   omitted if that ever happens.
		headers := amqp.Table{retriesHeaderName: payload.Retries}
		chanLock.Lock()
		defer chanLock.Unlock()
		err = rmq.Channel.Publish(
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
