package rmqhttp

import (
	"fmt"
)

// https://stackoverflow.com/a/52826567
func ToInt(anint interface{}) (int, error) {
	var i int

	switch t := anint.(type) {
	case int:
		i = t
	case int8:
		i = int(t)
	case int16:
		i = int(t)
	case int32:
		i = int(t)
	case int64:
		i = int(t)
	case float32:
		i = int(t)
	case float64:
		i = int(t)
	case uint8:
		i = int(t)
	case uint16:
		i = int(t)
	case uint32:
		i = int(t)
	case uint64:
		i = int(t)
	default:
		return i, fmt.Errorf("Failed to convert %T to int.", anint)
	}

	return i, nil
}
