package config

import (
	"fmt"

	"szuro.net/zms/internal/filter"
	"szuro.net/zms/internal/observer"
	"szuro.net/zms/internal/plugin"
)

type Target struct {
	Name              string
	Type              string
	Connection        string
	OfflineBufferTime int64         `yaml:"offline_buffer_time"` // Time in hours to keep offline buffer
	TagFilter         filter.Filter `yaml:"tag_filters"`
	Source            []string
	Options           map[string]string
}

func (t *Target) ToObserver() (obs observer.Observer, err error) {
	// Check if this is a plugin type
	if plugin.IsPluginType(t.Type) {
		pluginName := plugin.ExtractPluginName(t.Type)
		pluginObserver, err := plugin.GetRegistry().CreateObserver(pluginName)
		if err != nil {
			return nil, fmt.Errorf("failed to create plugin observer %s: %w", pluginName, err)
		}

		// Initialize the plugin
		if err := pluginObserver.Initialize(t.Connection, t.Options); err != nil {
			return nil, fmt.Errorf("failed to initialize plugin observer %s: %w", pluginName, err)
		}

		// Wrap the plugin observer to implement our internal Observer interface
		obs = &PluginObserverWrapper{
			pluginObserver: pluginObserver,
			targetName:     t.Name,
		}
	} else {
		// Handle built-in observers
		switch t.Type {
		case "print":
			obs = observer.NewPrint(t.Name, t.Connection)
		case "azuretable":
			// obs, err = observer.NewAzureTable(t.Name, t.Connection)
		case "pushgateway":
			obs, err = observer.NewPushGatewayManager(t.Name, t.Connection)
		case "gcp_cloud_monitor":
			obs, err = observer.NewCloudMonitor(t.Name, t.Connection)
		case "psql":
			// obs, err = observer.NewPSQL(t.Name, t.Connection, t.Options)
		default:
			return nil, fmt.Errorf("target type not supported: %s", t.Type)
		}
	}

	if obs == nil {
		return nil, fmt.Errorf("failed to create observer")
	}

	filter := t.TagFilter
	filter.Activate()
	obs.SetFilter(filter)
	obs.PrepareMetrics(t.Source)

	return obs, err
}
