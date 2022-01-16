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

func getManagementConnectionString() string {
	s := os.Getenv("RABBITMQ_MANAGEMENT_CONNECTION_STRING")
	if s == "" {
		s = "http://guest:guest@rabbitmq:15672/"
		log.Warn("Using default RMQ management connection string")
	}

	return s
}
