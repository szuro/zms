package plugin

import (
	"slices"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/pkg/filter"

	"szuro.net/zms/pkg/zbx"
)

// BaseObserverImpl provides the concrete implementation of BaseObserver.
// This struct should be embedded in plugin implementations to provide access
// to core ZMS functionality including filtering, metrics, and buffering.
//
// Plugins can access the public Monitor field to update metrics, and use
// methods like EvaluateFilter to apply configured filters.
type BaseObserverImpl struct {
	// name is the configured name of this observer instance
	name string

	// plugin is the plugin type unique identifier
	plugin string

	// Monitor provides access to Prometheus metrics for tracking operations.
	// Plugins should use these counters to report success/failure statistics.
	Monitor observerMetrics

	// localFilter handles tag-based filtering for this observer
	Filter filter.Filter

	// Buffer provides offline data storage capability
	Buffer ZMSBuffer

	// enabledExports tracks which export types this observer handles
	enabledExports []string
}

// observerMetrics holds the Prometheus metrics for an observer.
// These counters track the number of successful and failed operations
// for each export type, providing observability into plugin performance.
type observerMetrics struct {
	// HistoryValuesSent tracks successful history record processing
	HistoryValuesSent prometheus.Counter

	// HistoryValuesFailed tracks failed history record processing
	HistoryValuesFailed prometheus.Counter

	// TrendsValuesSent tracks successful trend record processing
	TrendsValuesSent prometheus.Counter

	// TrendsValuesFailed tracks failed trend record processing
	TrendsValuesFailed prometheus.Counter

	// EventsValuesSent tracks successful event record processing
	EventsValuesSent prometheus.Counter

	// EventsValuesFailed tracks failed event record processing
	EventsValuesFailed prometheus.Counter
}

// NewBaseObserver creates a new BaseObserver instance.
// This function is typically not needed by plugins since they should embed
// BaseObserverImpl directly in their struct.
//
// Parameters:
//   - name: the configured name for this observer instance
//   - plugin: the plugin type identifier
func NewBaseObserver(name, plugin string) *BaseObserverImpl {
	return &BaseObserverImpl{
		name:   name,
		plugin: plugin,
	}
}

// GetName returns the configured name of this observer instance.
// Implements the BaseObserver interface.
func (b *BaseObserverImpl) GetName() string {
	return b.name
}

// SetName sets the name of this observer instance.
// This is called by ZMS to assign the configured target name.
// Implements the BaseObserver interface.
func (b *BaseObserverImpl) SetName(name string) {
	b.name = name
}

// InitBuffer initializes the offline buffer for this observer.
// The buffer is used to store data when the target is unavailable,
// providing reliability through temporary persistence.
// Implements the BaseObserver interface.
func (b *BaseObserverImpl) InitBuffer(bufferPath string, ttl int64) {
	b.Buffer = ZMSDefaultBuffer{}
	b.Buffer.InitBuffer(bufferPath, ttl)
}

// PrepareFilter creates a filter instance from raw configuration data.
// The filter is used to determine which data should be processed based on tag matching rules.
// Returns a DefaultFilter with empty rules if rawFilter is nil, otherwise parses the
// configuration map to extract accepted and rejected tag patterns.
// Implements the BaseObserver interface.
func (b *BaseObserverImpl) PrepareFilter(rawFilter any) filter.Filter {
	f := &filter.DefaultFilter{}
	if rawFilter != nil {
		f = filter.NewDefaultFilter(rawFilter.(map[string]any))
	}
	return f
}

// SetFilter configures the tag filter for this observer.
// The filter is used to determine which data should be processed
// by this observer based on tag matching rules.
// Implements the BaseObserver interface.
func (b *BaseObserverImpl) SetFilter(filter filter.Filter) {
	b.Filter = filter
}

// PrepareMetrics initializes Prometheus metrics for the specified export types.
// This sets up the metrics counters that plugins can use to track their operations.
// Implements the BaseObserver interface.
func (b *BaseObserverImpl) PrepareMetrics(exports []string) {
	b.enabledExports = exports
	b.initObserverMetrics()
}

// Cleanup releases resources held by the base observer.
// This includes closing the offline buffer and cleaning up any other resources.
// Plugins should call this method in their own Cleanup implementation.
// Implements the BaseObserver interface.
func (b *BaseObserverImpl) Cleanup() {
	if b.Buffer != nil {
		b.Buffer.Cleanup()
	}
}

// EvaluateFilter checks if the given tags pass the configured filter.
// Plugins should use this method to apply filtering before processing data.
// Returns true if the data should be processed, false if it should be filtered out.
// func (b *BaseObserverImpl) EvaluateFilter(tags []zbx.Tag) bool {
// 	return b.localFilter.EvaluateFilter(tags)
// }

// SaveHistory is not implemented in BaseObserverImpl.
// Plugins must implement this method to process history data.
// This method will panic if called, forcing plugins to provide their own implementation.
func (b *BaseObserverImpl) SaveHistory(h []zbx.History) bool {
	panic("SaveHistory is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveTrends is not implemented in BaseObserverImpl.
// Plugins must implement this method to process trend data.
// This method will panic if called, forcing plugins to provide their own implementation.
func (b *BaseObserverImpl) SaveTrends(t []zbx.Trend) bool {
	panic("SaveTrends is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveEvents is not implemented in BaseObserverImpl.
// Plugins must implement this method to process event data.
// This method will panic if called, forcing plugins to provide their own implementation.
func (b *BaseObserverImpl) SaveEvents(e []zbx.Event) bool {
	panic("SaveEvents is not implemented in baseObserver, please implement it in the derived observer type")
}

// initObserverMetrics initializes the Prometheus metrics for this observer.
// This method is called automatically by PrepareMetrics and sets up counters
// for tracking successful and failed operations for each enabled export type.
// The metrics include labels for target_name, plugin_name, and export_type.
func (b *BaseObserverImpl) initObserverMetrics() {
	if slices.Contains(b.enabledExports, zbx.HISTORY) {
		b.Monitor.HistoryValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": b.name, "plugin_name": b.plugin, "export_type": zbx.HISTORY},
		})

		b.Monitor.HistoryValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": b.name, "plugin_name": b.plugin, "export_type": zbx.HISTORY},
		})
	}
	if slices.Contains(b.enabledExports, zbx.TREND) {
		b.Monitor.TrendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": b.name, "plugin_name": b.plugin, "export_type": zbx.TREND},
		})

		b.Monitor.TrendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": b.name, "plugin_name": b.plugin, "export_type": zbx.TREND},
		})
	}
	if slices.Contains(b.enabledExports, zbx.EVENT) {
		b.Monitor.EventsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": b.name, "plugin_name": b.plugin, "export_type": zbx.EVENT},
		})

		b.Monitor.EventsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": b.name, "plugin_name": b.plugin, "export_type": zbx.EVENT},
		})
	}
}
