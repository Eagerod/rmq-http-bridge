package rmqhttp

import (
	"encoding/json"
	"errors"
)

// Desribes the primary payload of the system.
//
// Endpoint:     URL where the content will be sent.
// Content:      Payload to send in HTTP request.
// Base64Decode: Whether or not the service needs to decode the given content
// before sending it.
//
// Retries:      Number of retries before pushing to the DLX.
// Defaults to 2; maximum 9.
//
// Headers:		 Map of headers to send to the server.
// If the server needs a content type to interpret the payload,
// include it here.
//
// Backoff:      Minimum number of seconds for the first of the expoentially
// backing off retries.
// Defaults to 1 second.
//
// Timeout:      Number of seconds to wait before timing out the HTTP request.
// Defaults to 60 seconds; maximum 3600 seconds.
type rmqPayload struct {
	Endpoint     string
	Content      string
	Base64Decode bool
	Retries      int
	Headers      map[string]string
	Backoff      int
	Timeout      int
}

func NewRMQPayload(bytes []byte) (*rmqPayload, error) {
	payload := rmqPayload{Retries: 2, Backoff: 1, Timeout: 60}
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return nil, errors.New("invalid JSON")
	}

	if payload.Endpoint == "" {
		return nil, errors.New("no endpoint given")
	}

	if payload.Retries < 0 || payload.Retries > 9 {
		return nil, errors.New("retries not within (0, 9)")
	}

	return &payload, nil
}
