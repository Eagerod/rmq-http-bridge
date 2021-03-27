package rmqhttp

import (
	"encoding/json"
)

type JsonError struct {
	Error   string
	Message string
}

func NewJsonError(err, message string) *JsonError {
	return &JsonError{Error: err, Message: message}
}

func (j *JsonError) Json() []byte {
	json, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}

	return json
}
