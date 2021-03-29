package rmqhttp

import (
	"math"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestDelatRoutingKey(t *testing.T) {
	var tests = []struct {
		name   string
		delay  int64
		dest   string
		output string
	}{
		{"No Delay", 0, "q", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.q"},
		{"1 Delay", 1, "q", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.q"},
		{"10 Delay", 10, "q", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.1.0.q"},
		{"100 Delay", 100, "q", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1.1.0.0.1.0.0.q"},
		{"Max Delay", int64(math.Pow(2, DelayInfrastructureBitCount)) - 1, "q", "1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.1.q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.output, DelayRoutingKey(tt.dest, tt.delay))
		})
	}
}
