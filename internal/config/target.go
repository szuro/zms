package config

import (
	"fmt"

	"szuro.net/zms/internal/plugin"
	plug "szuro.net/zms/pkg/plugin"
)

type Target struct {
	Name              string
	PluginName        string `yaml:"type"`
	Connection        string
	OfflineBufferTime int64 `yaml:"offline_buffer_time"` // Time in hours to keep offline buffer
	RawFilter         any   `yaml:"filter"`
	Source            []string
	Options           map[string]string
}

func (t *Target) ToObserver() (obs plug.Observer, err error) {
	obs, err = plugin.GetRegistry().CreateObserver(t.PluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin observer %s: %w", t.PluginName, err)
	}

	if err := obs.Initialize(t.Connection, t.Options); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin observer %s: %w", t.PluginName, err)
	}

	// to initialize
	obs.SetName(t.Name)
	obs.SetFilter(t.RawFilter)
	obs.PrepareMetrics(t.Source)

	return obs, err
}
