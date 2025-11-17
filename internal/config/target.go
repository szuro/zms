package config

import (
	"fmt"

	"zms.szuro.net/internal/plugin"
	"zms.szuro.net/pkg/filter"
)

type Target struct {
	UniqueName        string `yaml:"name"`
	PluginBinaryName  string `yaml:"type"`
	Connection        string
	OfflineBufferTime int64               `yaml:"offline_buffer_time"` // Time in hours to keep offline buffer
	Filter            filter.FilterConfig `yaml:"filter"`
	Source            []string
	Options           map[string]string
}

func (t *Target) ToObserver(config ZMSConf) (obs Observer, err error) {
	// Try gRPC registry first (new plugin system)
	if _, exists := plugin.GetGRPCRegistry().GetPlugin(t.PluginBinaryName); exists {
		// Create gRPC observer
		grpcObs, err := t.ToGRPCObserver(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC plugin observer %s: %w", t.PluginBinaryName, err)
		}

		// Wrap in adapter that implements plug.Observer interface
		return grpcObs, nil
	}

	return obs, err
}
