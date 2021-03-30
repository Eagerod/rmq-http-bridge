package rmqhttp

import (
	"os"
)

import (
	log "github.com/sirupsen/logrus"
)

func getConnectionString() string {
	s := os.Getenv("RABBITMQ_CONNECTION_STRING")
	if s == "" {
		s = "amqp://guest:guest@rabbitmq:5672/"
		log.Warn("Using default RMQ connection string")
	}

	return s
}
