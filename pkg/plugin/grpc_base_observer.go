package plugin

import (
	"context"
	"log/slog"

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
	b.Filter = b.createFilterFromProto(req.Filter)

	return &proto.InitializeResponse{Success: true}, nil
}

// FilterHistory applies the configured filter to history data.
// Returns only the history records that pass the filter.
func (b *BaseObserverGRPC) FilterHistory(history []*proto.History) []zbx.History {
	zbxHistory := protoHistoryToZbx(history)
	return b.Filter.FilterHistory(zbxHistory)
}

// FilterTrends applies the configured filter to trend data.
// Returns only the trend records that pass the filter.
func (b *BaseObserverGRPC) FilterTrends(trends []*proto.Trend) []zbx.Trend {
	zbxTrends := protoTrendsToZbx(trends)
	return b.Filter.FilterTrends(zbxTrends)
}

// FilterEvents applies the configured filter to event data.
// Returns only the event records that pass the filter.
func (b *BaseObserverGRPC) FilterEvents(events []*proto.Event) []zbx.Event {
	zbxEvents := protoEventsToZbx(events)
	return b.Filter.FilterEvents(zbxEvents)
}

// createFilterFromProto converts a proto.FilterConfig to a filter.Filter instance.
func (b *BaseObserverGRPC) createFilterFromProto(filter_ *proto.Filter) filter.Filter {
	if filter_ == nil {
		return &filter.DefaultFilter{}
	}

	filterConfig := filter.FilterConfig{Accepted: filter_.Accepted, Rejected: filter_.Rejected}
	//TODO: handle filter_.Type if custom filters are implemented in the future
	return filter.NewTagFilter(filterConfig)
}
