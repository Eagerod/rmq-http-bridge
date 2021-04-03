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
//               before sending it.
// Retries:      Number of retries before pushing to the DLX.
//				 Defaults to 2; maximum 9.
// Headers:		 Map of headers to send to the server.
//               If the server needs a content type to interpret the payload,
//               include it here.
type rmqPayload struct {
	Endpoint     string
	Content      string
	Base64Decode bool
	Retries      int
	Headers      map[string]string
}

func NewRMQPayload(bytes []byte) (*rmqPayload, error) {
	payload := rmqPayload{Retries: 2}
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return nil, errors.New("Invalid JSON")
	}

	if payload.Endpoint == "" {
		return nil, errors.New("No endpoint given")
	}

	if payload.Retries < 0 || payload.Retries > 9 {
		return nil, errors.New("Retries not within (0, 9)")
	}

	return &payload, nil
}
