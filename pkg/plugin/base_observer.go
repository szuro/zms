package plugin

import (
	"slices"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/internal/filter"

	"szuro.net/zms/pkg/zbx"
)

// BaseObserverImpl provides the concrete implementation of BaseObserver
// This gives plugins access to baseObserver functionality
type BaseObserverImpl struct {
	name           string
	observerType   string
	Monitor        observerMetrics
	localFilter    filter.Filter
	buffer         ZMSBuffer
	enabledExports []string
}

// observerMetrics holds the Prometheus metrics for an observer
type observerMetrics struct {
	HistoryValuesSent   prometheus.Counter
	HistoryValuesFailed prometheus.Counter
	TrendsValuesSent    prometheus.Counter
	TrendsValuesFailed  prometheus.Counter
	EventsValuesSent    prometheus.Counter
	EventsValuesFailed  prometheus.Counter
}

// NewBaseObserver creates a new BaseObserver instance
func NewBaseObserver(name, observerType string) *BaseObserverImpl {
	return &BaseObserverImpl{
		name:         name,
		observerType: observerType,
	}
}

// GetName returns the observer name
func (b *BaseObserverImpl) GetName() string {
	return b.name
}

// SetName sets the observer name
func (b *BaseObserverImpl) SetName(name string) {
	b.name = name
}

// InitBuffer initializes the offline buffer
func (b *BaseObserverImpl) InitBuffer(bufferPath string, ttl int64) {
	b.buffer = ZMSDefaultBuffer{}
	b.buffer.InitBuffer(bufferPath, ttl)
}

// SetFilter sets the local filter
func (b *BaseObserverImpl) SetFilter(filter filter.Filter) {
	b.localFilter = filter
}

// PrepareMetrics initializes Prometheus metrics
func (b *BaseObserverImpl) PrepareMetrics(exports []string) {
	b.enabledExports = exports
	b.initObserverMetrics()
}

// Cleanup releases resources
func (b *BaseObserverImpl) Cleanup() {
	b.buffer.Cleanup()
}

// EvaluateFilter checks if data passes the local filter
func (b *BaseObserverImpl) EvaluateFilter(tags []zbx.Tag) bool {
	return b.localFilter.EvaluateFilter(tags)
}

func (b *BaseObserverImpl) SaveHistory(h []zbx.History) bool {
	panic("SaveHistory is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveTrends processes and saves a slice of zbx.Trend objects using a generic saving function.
// It applies a local filter to each trend's tags, serializes trends for storage, and manages buffering
// with offline TTL support. Returns true if the save operation succeeds.
func (b *BaseObserverImpl) SaveTrends(t []zbx.Trend) bool {
	panic("SaveTrends is not implemented in baseObserver, please implement it in the derived observer type")
}

// SaveEvents processes and saves a slice of zbx.Event objects using a generic saving function.
// It applies a local filter to each event's tags, executes a custom event function, and manages buffering
// with offline TTL support. Events are serialized to and from byte slices for storage.
// Returns true if the events were successfully saved.
func (b *BaseObserverImpl) SaveEvents(e []zbx.Event) bool {
	panic("SaveEvents is not implemented in baseObserver, please implement it in the derived observer type")
}

// // GetMetrics returns the observer metrics for external access
// func (b *BaseObserverImpl) GetMetrics() (exportsSent, exportsFailed prometheus.Counter) {
// 	return b.monitor.exportsSent, b.monitor.exportsFailed
// }

// Private implementation methods

func (b *BaseObserverImpl) initObserverMetrics() {
	if slices.Contains(b.enabledExports, zbx.HISTORY) {
		b.Monitor.HistoryValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": b.name, "target_type": b.observerType, "export_type": zbx.HISTORY},
		})

		b.Monitor.HistoryValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": b.name, "target_type": b.observerType, "export_type": zbx.HISTORY},
		})
	}
	if slices.Contains(b.enabledExports, zbx.TREND) {
		b.Monitor.TrendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": b.name, "target_type": b.observerType, "export_type": zbx.TREND},
		})

		b.Monitor.TrendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": b.name, "target_type": b.observerType, "export_type": zbx.TREND},
		})
	}
	if slices.Contains(b.enabledExports, zbx.EVENT) {
		b.Monitor.EventsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": b.name, "target_type": b.observerType, "export_type": zbx.EVENT},
		})

		b.Monitor.EventsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": b.name, "target_type": b.observerType, "export_type": zbx.EVENT},
		})
	}
}
