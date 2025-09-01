package zbx

import (
	"testing"
	"time"
)

func TestGetFailoverDelay(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "Valid input with delay",
			input:    "Some text here. Failover delay: 30 seconds",
			expected: time.Duration(30) * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFailoverDelay(tt.input)
			if result != tt.expected {
				t.Errorf("GetFailoverDelay() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
