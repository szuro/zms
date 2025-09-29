package main

import (
	"fmt"
	"io"
	"os"

	"szuro.net/zms/pkg/filter"
	zbxpkg "szuro.net/zms/pkg/zbx"

	"szuro.net/zms/pkg/plugin"
)

const (
	STDOUT      = "stdout"
	STDERR      = "stderr"
	PLUGIN_NAME = "log_print"
)

// LogFilter implements filter.Filter interface for log_print plugin
// It filters history items to only accept LOG type entries
type LogFilter struct{}

// AcceptHistory returns true only for LOG type history items
func (lf *LogFilter) AcceptHistory(h zbxpkg.History) bool {
	return h.Type == zbxpkg.LOG
}

// AcceptTrend rejects all trend items (not supported by this plugin)
func (lf *LogFilter) AcceptTrend(t zbxpkg.Trend) bool {
	return false
}

// AcceptEvent rejects all event items (not supported by this plugin)
func (lf *LogFilter) AcceptEvent(e zbxpkg.Event) bool {
	return false
}

// FilterHistory filters history records to only include LOG type items
func (lf *LogFilter) FilterHistory(h []zbxpkg.History) []zbxpkg.History {
	accepted := make([]zbxpkg.History, 0, len(h))
	for _, history := range h {
		if lf.AcceptHistory(history) {
			accepted = append(accepted, history)
		}
	}
	return accepted
}

// FilterTrends returns empty array (trends not supported)
func (lf *LogFilter) FilterTrends(t []zbxpkg.Trend) []zbxpkg.Trend {
	return make([]zbxpkg.Trend, 0)
}

// FilterEvents returns empty array (events not supported)
func (lf *LogFilter) FilterEvents(e []zbxpkg.Event) []zbxpkg.Event {
	return make([]zbxpkg.Event, 0)
}

// PluginInfo provides metadata about this plugin
var PluginInfo = plugin.PluginInfo{
	Name:        PLUGIN_NAME,
	Version:     "1.0.0",
	Description: "Filters and outputs LOG type Zabbix history to stdout/stderr",
	Author:      "ZMS Example",
}

// Print is the observer implementation for log output
type Print struct {
	plugin.BaseObserverImpl
	out io.Writer
}

// NewObserver creates a new Print observer instance
// This is the required factory function called by the plugin loader
func NewObserver() plugin.Observer {
	return &Print{}
}

// Initialize configures the plugin with connection string and options
// connection: "stdout" or "stderr" to determine output destination
// options: additional configuration (currently unused)
func (p *Print) Initialize(connection string, options map[string]string) error {
	switch connection {
	case STDERR:
		p.out = os.Stderr
	default:
		p.out = os.Stdout
	}

	return nil
}

// GetType returns the plugin type identifier
func (p *Print) GetType() string {
	return PLUGIN_NAME
}

// PrepareFilter creates a LogFilter that only accepts LOG type history items
func (p *Print) PrepareFilter(rawFilter any) filter.Filter {
	return &LogFilter{}
}

// SaveHistory processes and outputs LOG type history items
// Only history items that pass the LogFilter (type == LOG) are output
func (p *Print) SaveHistory(h []zbxpkg.History) bool {
	history := p.Filter.FilterHistory(h)
	for _, H := range history {
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %v", H.Host.Host, H.Name, H.Clock, H.Value)
		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			p.Monitor.HistoryValuesFailed.Inc()
		}
		p.Monitor.HistoryValuesSent.Inc()
	}
	return true
}

// SaveTrends and SaveEvents are not implemented - they will panic if called
// This plugin only supports history (LOG type) processing
