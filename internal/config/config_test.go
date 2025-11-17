package config

import (
	"testing"

	"github.com/stretchr/testify/require"
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
			config := ZMSConf{BufferSize: tt.input}
			config.setBuffer()
			result := config.BufferSize
			if result != tt.expected {
				t.Errorf("setBuffer() with BufferSize=%d = %d; want %d", tt.input, result, tt.expected)
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
			config := ZMSConf{Mode: tt.input}
			config.setMode()
			result := config.Mode
			if result != tt.expected {
				t.Errorf("setMode() with Mode=%s = %s; want %s", tt.input, result, tt.expected)
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
			config := ZMSConf{Http: HTTPConf{ListenPort: tt.input}}
			config.setPort()
			result := config.Http.ListenPort
			if result != tt.expected {
				t.Errorf("setPort() with ListenPort=%d = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestZMSConf_setOfflineBuffers(t *testing.T) {
	tests := []struct {
		name     string
		targets  []Target
		expected []int64
	}{
		{
			name:     "All positive values",
			targets:  []Target{{OfflineBufferTime: 10}, {OfflineBufferTime: 5}},
			expected: []int64{10, 5},
		},
		{
			name:     "All negative values",
			targets:  []Target{{OfflineBufferTime: -1}, {OfflineBufferTime: -100}},
			expected: []int64{0, 0},
		},
		{
			name:     "Mixed values",
			targets:  []Target{{OfflineBufferTime: -5}, {OfflineBufferTime: 0}, {OfflineBufferTime: 7}},
			expected: []int64{0, 0, 7},
		},
		{
			name:     "Empty targets",
			targets:  []Target{},
			expected: []int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ZMSConf{
				Targets: tt.targets,
			}
			conf.setOfflineBuffers()
			for i, target := range conf.Targets {
				require.Equal(t, tt.expected[i], target.OfflineBufferTime)
			}
		})
	}
}
