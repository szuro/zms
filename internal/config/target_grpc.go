package config

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/internal/logger"
	"szuro.net/zms/internal/plugin"
	pluginPkg "szuro.net/zms/pkg/plugin"
	"szuro.net/zms/pkg/zbx"
	"szuro.net/zms/proto"
)

type Observer interface {
	// Cleanup releases any resources held by the observer.
	// Called when the observer is being shut down or removed.
	Cleanup()

	// GetName returns the configured name of this observer instance.
	GetName() string

	// Data processing methods - implement these for your plugin logic

	// SaveHistory processes and saves history data.
	// Returns true if processing was successful, false otherwise.
	SaveHistory(h []zbx.History) bool

	// SaveTrends processes and saves trend data.
	// Returns true if processing was successful, false otherwise.
	SaveTrends(t []zbx.Trend) bool

	// SaveEvents processes and saves event data.
	// Returns true if processing was successful, false otherwise.
	SaveEvents(e []zbx.Event) bool
}

// GRPCObserver wraps a gRPC plugin observer for use in ZMS.
type GRPCObserver struct {
	client     proto.ObserverServiceClient
	pluginName string
	name       string
	// monitor provides access to Prometheus metrics for tracking operations.
	// Plugins should use these counters to report success/failure statistics.
	monitor        observerMetrics
	enabledExports []string
}

// ToGRPCObserver creates a gRPC observer from the target configuration.
// This initializes the gRPC plugin, sends configuration, and returns a wrapper.
func (t *Target) ToGRPCObserver(config ZMSConf) (*GRPCObserver, error) {
	// Create observer from gRPC plugin
	client, _, err := plugin.GetGRPCRegistry().CreateObserver(t.PluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC plugin observer %s: %w", t.PluginName, err)
	}

	// Prepare filter config
	// var filters []*proto.Filter
	filterConfig := &proto.Filter{
		Type:     proto.FilterType(proto.FilterType_TAG),
		Accepted: t.Filter.Accepted,
		Rejected: t.Filter.Rejected,
	}

	// Convert export types
	exports := make([]proto.ExportType, 0, len(t.Source))
	for _, exportType := range t.Source {
		exports = append(exports, pluginPkg.StringToExportType(exportType))
	}

	// Initialize the plugin
	initReq := &proto.InitializeRequest{
		Name:       t.Name,
		Connection: t.Connection,
		Options:    t.Options,
		Exports:    exports,
		Filter:     filterConfig,
	}

	resp, err := client.Initialize(context.Background(), initReq)
	if err != nil {
		client.Cleanup(context.Background(), &proto.CleanupRequest{})
		return nil, fmt.Errorf("failed to initialize gRPC plugin observer %s: %w", t.PluginName, err)
	}

	if !resp.Success {
		client.Cleanup(context.Background(), &proto.CleanupRequest{})
		return nil, fmt.Errorf("plugin initialization failed: %s", resp.Error)
	}

	obs := &GRPCObserver{
		client:         client,
		pluginName:     t.PluginName,
		name:           t.Name,
		enabledExports: t.Source,
	}
	obs.initObserverMetrics()
	return obs, nil
}

// GetClient returns the gRPC client for this observer.
func (o *GRPCObserver) GetClient() proto.ObserverServiceClient {
	return o.client
}

// GetName returns the configured name of this observer.
func (o *GRPCObserver) GetName() string {
	return o.name
}

// GetPluginName returns the plugin type name.
func (o *GRPCObserver) GetPluginName() string {
	return o.pluginName
}

// Cleanup releases resources by calling the gRPC plugin's Cleanup method.
func (o *GRPCObserver) Cleanup() {
	if o != nil {
		ctx := context.Background()
		req := &proto.CleanupRequest{}
		_, err := o.client.Cleanup(ctx, req)
		if err != nil {
			logger.Error("Failed to cleanup gRPC plugin",
				slog.String("plugin", o.pluginName),
				slog.Any("error", err))
		}
	}
}

// interfaceSliceToStringSlice converts []interface{} to []string.
func interfaceSliceToStringSlice(slice []any) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(map[string]any); ok {
			S := fmt.Sprintf("%v", s["tag"]) + ":" + fmt.Sprintf("%v", s["value"])
			result = append(result, S)
		}
	}
	return result
}

func (o *GRPCObserver) SaveHistory(h []zbx.History) bool {
	ctx := context.Background()

	// Convert zbx.History to proto.History
	protoHistory := make([]*proto.History, 0, len(h))
	for _, hist := range h {
		protoHistory = append(protoHistory, pluginPkg.ZbxHistoryToProto(&hist))
	}

	req := &proto.SaveHistoryRequest{
		History: protoHistory,
	}

	resp, err := o.client.SaveHistory(ctx, req)
	if err != nil {
		logger.Error("Failed to save history via gRPC plugin",
			slog.String("plugin", o.pluginName),
			slog.Any("error", err))
		return false
	}

	if !resp.Success {
		logger.Error("gRPC plugin reported failure saving history",
			slog.String("plugin", o.pluginName),
			slog.String("error", resp.Error))
		return false
	}
	o.monitor.HistoryValuesSent.Add(float64(resp.RecordsProcessed))
	o.monitor.HistoryValuesFailed.Add(float64(resp.RecordsFailed))

	return true
}

// SaveTrends processes trend data by converting to proto format and calling the gRPC method.
func (o *GRPCObserver) SaveTrends(t []zbx.Trend) bool {
	ctx := context.Background()

	// Convert zbx.Trend to proto.Trend
	protoTrends := make([]*proto.Trend, 0, len(t))
	for _, trend := range t {
		protoTrends = append(protoTrends, pluginPkg.ZbxTrendToProto(&trend))
	}

	req := &proto.SaveTrendsRequest{
		Trends: protoTrends,
	}

	resp, err := o.client.SaveTrends(ctx, req)
	if err != nil {
		logger.Error("Failed to save trends via gRPC plugin",
			slog.String("plugin", o.pluginName),
			slog.Any("error", err))
		return false
	}

	if !resp.Success {
		logger.Error("gRPC plugin reported failure saving trends",
			slog.String("plugin", o.pluginName),
			slog.String("error", resp.Error))
		return false
	}
	o.monitor.TrendsValuesSent.Add(float64(resp.RecordsProcessed))
	o.monitor.TrendsValuesFailed.Add(float64(resp.RecordsFailed))

	return true
}

// SaveEvents processes event data by converting to proto format and calling the gRPC method.
func (o *GRPCObserver) SaveEvents(e []zbx.Event) bool {
	ctx := context.Background()

	// Convert zbx.Event to proto.Event
	protoEvents := make([]*proto.Event, 0, len(e))
	for _, event := range e {
		protoEvents = append(protoEvents, pluginPkg.ZbxEventToProto(&event))
	}

	req := &proto.SaveEventsRequest{
		Events: protoEvents,
	}

	resp, err := o.client.SaveEvents(ctx, req)
	if err != nil {
		logger.Error("Failed to save events via gRPC plugin",
			slog.String("plugin", o.pluginName),
			slog.Any("error", err))
		return false
	}

	if !resp.Success {
		logger.Error("gRPC plugin reported failure saving events",
			slog.String("plugin", o.pluginName),
			slog.String("error", resp.Error))
		return false
	}
	o.monitor.EventsValuesSent.Add(float64(resp.RecordsProcessed))
	o.monitor.EventsValuesFailed.Add(float64(resp.RecordsFailed))

	return true
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

func (o *GRPCObserver) initObserverMetrics() {
	if slices.Contains(o.enabledExports, zbx.HISTORY) {
		o.monitor.HistoryValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": o.name, "plugin_name": o.pluginName, "export_type": zbx.HISTORY},
		})

		o.monitor.HistoryValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": o.name, "plugin_name": o.pluginName, "export_type": zbx.HISTORY},
		})
	}
	if slices.Contains(o.enabledExports, zbx.TREND) {
		o.monitor.TrendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": o.name, "plugin_name": o.pluginName, "export_type": zbx.TREND},
		})

		o.monitor.TrendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": o.name, "plugin_name": o.pluginName, "export_type": zbx.TREND},
		})
	}
	if slices.Contains(o.enabledExports, zbx.EVENT) {
		o.monitor.EventsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_operations_total",
			Help:        "Total number of shipping operations",
			ConstLabels: prometheus.Labels{"target_name": o.name, "plugin_name": o.pluginName, "export_type": zbx.EVENT},
		})

		o.monitor.EventsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "zms_shipping_errors_total",
			Help:        "Total number of shipping errors",
			ConstLabels: prometheus.Labels{"target_name": o.name, "plugin_name": o.pluginName, "export_type": zbx.EVENT},
		})
	}
}
