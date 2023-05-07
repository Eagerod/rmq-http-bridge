package rmqhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type HttpController struct {
	rmq   *RMQ
	queue *amqp.Queue

	managementUrl *url.URL
}

func NewHttpController() *HttpController {
	httpController := HttpController{
		rmq:   NewRMQ(),
		queue: nil,
	}
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

func (hc *HttpController) SetManagementConnectionString(mcs string) {
	u, err := url.Parse(mcs)
	if err != nil {
		panic(err)
	}

	hc.managementUrl = u
}

func (hc *HttpController) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header()["Content-Type"] = []string{"application/json"}
	w.Write(NewJsonError(http.StatusText(statusCode), message).Json())
}

func (hc *HttpController) HttpHandler(w http.ResponseWriter, r *http.Request) {
	requestStartTime := time.Now()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		hc.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := NewRMQPayload(body)
	if err != nil {
		hc.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Note: Retries stays in the body.
	// There may eventually be a need to rewrite the body; it can be
	//   omitted if that ever happens.
	headers := amqp.Table{
		retriesHeaderName:    payload.Retries,
		retryDelayHeaderName: payload.Backoff,
	}

	channel, err := hc.rmq.LockChannel()
	if err != nil {
		log.Error(err)
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

	requestDuration := time.Since(requestStartTime)
	log.Debugf("Published to queue: %s; %07d byte payload in %05dms to: %s",
		hc.queue.Name, len(payload.Content), requestDuration.Milliseconds(), payload.Endpoint)

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
		log.Error(err)
		hc.respondError(w, http.StatusInternalServerError, "Failed to inspect DLQ")
		return
	}

	if queueInspect.Messages != 0 {
		msg := fmt.Sprintf("DLQ has %d items", queueInspect.Messages)
		hc.respondError(w, http.StatusInternalServerError, msg)
		return
	}
}

func (hc *HttpController) StatsHandler(w http.ResponseWriter, r *http.Request) {
	if hc.managementUrl == nil {
		log.Error("Management API not configured")
		hc.respondError(w, http.StatusInternalServerError, "Management API not configured")
		return
	}

	// Currently only supports default vhost.
	u, _ := url.Parse(hc.managementUrl.String())
	u.Path = path.Join(u.Path, "api", "queues", "%%2F", hc.queue.Name)
	fullManagementUrl := u.String()

	client := &http.Client{}
	req, err := http.NewRequest("GET", fullManagementUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		hc.respondError(w, http.StatusInternalServerError, "Failed to get data")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	stats := rmqStats{}
	if err := json.Unmarshal(body, &stats); err != nil {
		log.Error(err)
		hc.respondError(w, http.StatusInternalServerError, "Failed to parse response")
		return
	}

	var statsRefined = struct {
		Messages int
		InRate   float32
		OutRate  float32
	}{
		stats.Messages,
		stats.MessageStats.PublishDetails.Rate,
		stats.MessageStats.AckDetails.Rate,
	}

	aJson, err := json.Marshal(statsRefined)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Header()["Content-Type"] = []string{"application/json"}
	w.Write(aJson)
}
