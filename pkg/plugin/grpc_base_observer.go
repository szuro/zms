package plugin

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/pkg/filter"
	"szuro.net/zms/pkg/zbx"
	"szuro.net/zms/proto"
)

// BaseObserverGRPC provides core functionality for gRPC-based observer plugins.
// Plugins should embed this struct to get access to filtering, metrics, and logging.
//
// Unlike the old plugin system, this base observer only handles:
// - Filtering (initialized during Initialize RPC)
// - Metrics (initialized during Initialize RPC)
// - Logging
//
// Buffer management is optional and can be implemented by plugins using the buffer package.
type BaseObserverGRPC struct {
	// Name is the configured name of this observer instance
	Name string

	// PluginName is the plugin type identifier
	PluginName string

	// Filter handles tag-based filtering for this observer
	Filter filter.Filter

	// Logger provides structured logging
	Logger *slog.Logger

	// Monitor provides access to Prometheus metrics
	Monitor observerMetrics

	// enabledExports tracks which export types this observer handles
	enabledExports []proto.ExportType
}

// NewBaseObserverGRPC creates a new BaseObserverGRPC instance.
// Plugins should call this in their constructor.
func NewBaseObserverGRPC() *BaseObserverGRPC {
	return &BaseObserverGRPC{
		Logger: slog.Default(),
	}
}

// Initialize handles common initialization tasks for all plugins.
// Plugins should call this method in their Initialize implementation
// before doing plugin-specific initialization.
//
// This method:
// - Stores the observer name and configuration
// - Sets up filtering based on the provided filter config
// - Initializes Prometheus metrics
func (b *BaseObserverGRPC) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	b.Name = req.Name
	b.enabledExports = req.Exports

	// Initialize filter
	if req.Filter != nil {
		b.Filter = b.createFilterFromProto(req.Filter)
	} else {
		b.Filter = &filter.DefaultFilter{}
	}

	// Initialize metrics
	b.initMetrics()

	return &proto.InitializeResponse{Success: true}, nil
}

// FilterHistory applies the configured filter to history data.
// Returns only the history records that pass the filter.
func (b *BaseObserverGRPC) FilterHistory(history []*proto.History) []zbx.History {
	if b.Filter == nil {
		return protoHistoryToZbx(history)
	}

	zbxHistory := protoHistoryToZbx(history)
	return b.Filter.FilterHistory(zbxHistory)
}

// FilterTrends applies the configured filter to trend data.
// Returns only the trend records that pass the filter.
func (b *BaseObserverGRPC) FilterTrends(trends []*proto.Trend) []zbx.Trend {
	if b.Filter == nil {
		return protoTrendsToZbx(trends)
	}

	zbxTrends := protoTrendsToZbx(trends)
	return b.Filter.FilterTrends(zbxTrends)
}

// FilterEvents applies the configured filter to event data.
// Returns only the event records that pass the filter.
func (b *BaseObserverGRPC) FilterEvents(events []*proto.Event) []zbx.Event {
	if b.Filter == nil {
		return protoEventsToZbx(events)
	}

	zbxEvents := protoEventsToZbx(events)
	return b.Filter.FilterEvents(zbxEvents)
}

// createFilterFromProto converts a proto.FilterConfig to a filter.Filter instance.
func (b *BaseObserverGRPC) createFilterFromProto(filterConfig *proto.FilterConfig) filter.Filter {
	if filterConfig == nil {
		return &filter.DefaultFilter{}
	}

	rawFilter := map[string]any{
		"accept": filterConfig.Accept,
		"reject": filterConfig.Reject,
	}

	return filter.NewDefaultFilter(rawFilter)
}

// initMetrics initializes Prometheus metrics for this observer.
func (b *BaseObserverGRPC) initMetrics() {
	for _, exportType := range b.enabledExports {
		exportName := exportTypeToString(exportType)

		switch exportType {
		case proto.ExportType_HISTORY:
			b.Monitor.HistoryValuesSent = promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_shipping_operations_total",
				Help:        "Total number of shipping operations",
				ConstLabels: prometheus.Labels{"target_name": b.Name, "plugin_name": b.PluginName, "export_type": exportName},
			})

			b.Monitor.HistoryValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_shipping_errors_total",
				Help:        "Total number of shipping errors",
				ConstLabels: prometheus.Labels{"target_name": b.Name, "plugin_name": b.PluginName, "export_type": exportName},
			})

		case proto.ExportType_TRENDS:
			b.Monitor.TrendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_shipping_operations_total",
				Help:        "Total number of shipping operations",
				ConstLabels: prometheus.Labels{"target_name": b.Name, "plugin_name": b.PluginName, "export_type": exportName},
			})

			b.Monitor.TrendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_shipping_errors_total",
				Help:        "Total number of shipping errors",
				ConstLabels: prometheus.Labels{"target_name": b.Name, "plugin_name": b.PluginName, "export_type": exportName},
			})

		case proto.ExportType_EVENTS:
			b.Monitor.EventsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_shipping_operations_total",
				Help:        "Total number of shipping operations",
				ConstLabels: prometheus.Labels{"target_name": b.Name, "plugin_name": b.PluginName, "export_type": exportName},
			})

			b.Monitor.EventsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
				Name:        "zms_shipping_errors_total",
				Help:        "Total number of shipping errors",
				ConstLabels: prometheus.Labels{"target_name": b.Name, "plugin_name": b.PluginName, "export_type": exportName},
			})
		}
	}
}

// exportTypeToString converts proto.ExportType to string.
func exportTypeToString(et proto.ExportType) string {
	switch et {
	case proto.ExportType_HISTORY:
		return zbx.HISTORY
	case proto.ExportType_TRENDS:
		return zbx.TREND
	case proto.ExportType_EVENTS:
		return zbx.EVENT
	default:
		return "unknown"
	}
}
