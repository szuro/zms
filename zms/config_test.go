package zms

import (
	"testing"
)

func TestSetBuffer(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"Zero", 0, 100},
		{"Non-Zero", 1337, 1337},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ZMSConf{}
			config.setBuffer(tt.input)
			result := config.BufferSize
			if result != tt.expected {
				t.Errorf("setPort(%d) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSetMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid Mode - FILE_MODE", FILE_MODE, FILE_MODE},
		{"Valid Mode - HTTP_MODE", HTTP_MODE, HTTP_MODE},
		{"Empty Mode", "", FILE_MODE},
		{"Random Mode", "FNORD", FILE_MODE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ZMSConf{}
			config.setMode(tt.input)
			result := config.Mode
			if result != tt.expected {
				t.Errorf("setMode(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSetPort(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"Zero Port", 0, 2020},
		{"Non-Zero Port", 8080, 8080},
		{"Another Non-Zero Port", 3000, 3000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ZMSConf{}
			config.setPort(tt.input)
			result := config.Http.ListenPort
			if result != tt.expected {
				t.Errorf("setPort(%d) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}
