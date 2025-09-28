package plugin

import (
	"szuro.net/zms/internal/filter"
	"szuro.net/zms/pkg/zbx"
)

// Observer defines the interface that all observer plugins must implement
type Observer interface {
	// Core observer methods
	Cleanup()
	GetName() string
	SetName(name string)
	InitBuffer(path string, ttl int64)
	SaveHistory(h []zbx.History) bool
	SaveTrends(t []zbx.Trend) bool
	SaveEvents(e []zbx.Event) bool
	SetFilter(filter filter.Filter)
	PrepareMetrics(exports []string)

	// Plugin-specific initialization
	Initialize(connection string, options map[string]string) error
}

// BaseObserver provides access to baseObserver functionality for plugins
type BaseObserver interface {
	// Base functionality that plugins can access
	GetName() string
	SetName(name string)
	InitBuffer(path string, ttl int64)
	SetFilter(filter filter.Filter)
	PrepareMetrics(exports []string)
	Cleanup()
}

// PluginFactory is the function signature that plugins must export
// This function will be called to create new instances of the plugin
type PluginFactory func() Observer

// PluginInfo contains metadata about the plugin
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
}
