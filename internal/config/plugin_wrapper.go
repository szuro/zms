package config

import (
	"szuro.net/zms/internal/filter"
	pluginPkg "szuro.net/zms/pkg/plugin"
	"szuro.net/zms/pkg/zbx"
)

// PluginObserverWrapper wraps a plugin observer to implement the internal Observer interface
type PluginObserverWrapper struct {
	pluginObserver pluginPkg.Observer
	targetName     string
}

// Cleanup delegates to the plugin observer
func (w *PluginObserverWrapper) Cleanup() {
	w.pluginObserver.Cleanup()
}

// GetName delegates to the plugin observer
func (w *PluginObserverWrapper) GetName() string {
	return w.pluginObserver.GetName()
}

// SetName delegates to the plugin observer
func (w *PluginObserverWrapper) SetName(name string) {
	w.pluginObserver.SetName(name)
}

// InitBuffer delegates to the plugin observer
func (w *PluginObserverWrapper) InitBuffer(path string, ttl int64) {
	w.pluginObserver.InitBuffer(path, ttl)
}

// SaveHistory delegates to the plugin observer
func (w *PluginObserverWrapper) SaveHistory(h []zbx.History) bool {
	return w.pluginObserver.SaveHistory(h)
}

// SaveTrends delegates to the plugin observer
func (w *PluginObserverWrapper) SaveTrends(t []zbx.Trend) bool {
	return w.pluginObserver.SaveTrends(t)
}

// SaveEvents delegates to the plugin observer
func (w *PluginObserverWrapper) SaveEvents(e []zbx.Event) bool {
	return w.pluginObserver.SaveEvents(e)
}

// SetFilter delegates to the plugin observer
func (w *PluginObserverWrapper) SetFilter(filter filter.Filter) {
	w.pluginObserver.SetFilter(filter)
}

// PrepareMetrics delegates to the plugin observer
func (w *PluginObserverWrapper) PrepareMetrics(exports []string) {
	w.pluginObserver.PrepareMetrics(exports)
}
