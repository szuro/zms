package main

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/go-plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"

	pluginPkg "zms.szuro.net/pkg/plugin"
	"zms.szuro.net/pkg/proto"
	zbxpkg "zms.szuro.net/pkg/zbx"
)

const (
	PLUGIN_NAME = "prometheus_pushgateway"
)

// PrometheusPushgatewayPlugin implements the gRPC observer interface
type PrometheusPushgatewayPlugin struct {
	proto.UnimplementedObserverServiceServer
	pluginPkg.BaseObserverGRPC
	gatewayURL string
	jobName    string
	registry   *prometheus.Registry
}

// NewPrometheusPushgatewayPlugin creates a new plugin instance
func NewPrometheusPushgatewayPlugin() *PrometheusPushgatewayPlugin {
	return &PrometheusPushgatewayPlugin{
		BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
	}
}

// Initialize configures the plugin with settings from main application
func (p *PrometheusPushgatewayPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	// Call base initialization to handle common setup
	resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
	if err != nil {
		return resp, err
	}

	// Set plugin name for metrics
	p.PluginName = PLUGIN_NAME

	// Configure Pushgateway
	p.gatewayURL = req.Connection
	p.jobName = req.Options["job_name"]
	if p.jobName == "" {
		p.jobName = "zms_export"
	}

	p.registry = prometheus.NewRegistry()

	p.Logger.Info("Prometheus Pushgateway plugin initialized",
		"gateway_url", p.gatewayURL,
		"job_name", p.jobName,
		"name", req.Name)

	return &proto.InitializeResponse{Success: true}, nil
}

// SaveHistory processes history data
func (p *PrometheusPushgatewayPlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
	// Filter history entries
	history := p.FilterHistory(req.History)

	processedCount := int64(0)
	failedCount := int64(0)

	for _, H := range history {
		// Only handle numeric values
		if H.Type != zbxpkg.FLOAT && H.Type != zbxpkg.UNSIGNED {
			continue
		}

		// Create a gauge metric for each history item
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zabbix_history_value",
			Help: "Zabbix history value",
			ConstLabels: prometheus.Labels{
				"host":   H.Host.Host,
				"item":   H.Name,
				"itemid": strconv.Itoa(H.ItemID),
			},
		})

		// Set the value
		if value, ok := H.Value.(float64); ok {
			gauge.Set(value)
		} else if value, ok := H.Value.(int); ok {
			gauge.Set(float64(value))
		} else if value, ok := H.Value.(uint64); ok {
			gauge.Set(float64(value))
		}

		// Register and push
		p.registry.MustRegister(gauge)

		pusher := push.New(p.gatewayURL, p.jobName).
			Gatherer(p.registry).
			Grouping("instance", H.Host.Host)

		if err := pusher.Push(); err != nil {
			failedCount++
			p.Logger.Error("Failed to push to gateway", "error", err)
		} else {
			processedCount++
		}

		// Unregister for next iteration
		p.registry.Unregister(gauge)
	}

	return &proto.SaveResponse{
		Success:          failedCount == 0,
		RecordsProcessed: processedCount,
		RecordsFailed:    failedCount,
	}, nil
}

// SaveTrends processes trend data
func (p *PrometheusPushgatewayPlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
	// Filter trend entries
	trends := p.FilterTrends(req.Trends)

	processedCount := int64(0)
	failedCount := int64(0)

	for _, T := range trends {
		// Create gauge metrics for min, max, avg
		minGauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zabbix_trend_min",
			Help: "Zabbix trend minimum value",
			ConstLabels: prometheus.Labels{
				"host":   T.Host.Host,
				"item":   T.Name,
				"itemid": strconv.Itoa(T.ItemID),
			},
		})

		maxGauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zabbix_trend_max",
			Help: "Zabbix trend maximum value",
			ConstLabels: prometheus.Labels{
				"host":   T.Host.Host,
				"item":   T.Name,
				"itemid": strconv.Itoa(T.ItemID),
			},
		})

		avgGauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zabbix_trend_avg",
			Help: "Zabbix trend average value",
			ConstLabels: prometheus.Labels{
				"host":   T.Host.Host,
				"item":   T.Name,
				"itemid": strconv.Itoa(T.ItemID),
			},
		})

		minGauge.Set(T.Min)
		maxGauge.Set(T.Max)
		avgGauge.Set(T.Avg)

		// Register all gauges
		p.registry.MustRegister(minGauge, maxGauge, avgGauge)

		pusher := push.New(p.gatewayURL, p.jobName).
			Gatherer(p.registry).
			Grouping("instance", T.Host.Host)

		if err := pusher.Push(); err != nil {
			failedCount++
			p.Logger.Error("Failed to push trends to gateway", "error", err)
		} else {
			processedCount++
		}

		// Unregister for next iteration
		p.registry.Unregister(minGauge)
		p.registry.Unregister(maxGauge)
		p.registry.Unregister(avgGauge)
	}

	return &proto.SaveResponse{
		Success:          failedCount == 0,
		RecordsProcessed: processedCount,
		RecordsFailed:    failedCount,
	}, nil
}

// SaveEvents is not supported by this plugin - returns success with no-op
func (p *PrometheusPushgatewayPlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// Cleanup releases any resources held by the plugin
func (p *PrometheusPushgatewayPlugin) Cleanup(ctx context.Context, req *proto.CleanupRequest) (*proto.CleanupResponse, error) {
	p.Logger.Info("Cleaning up Prometheus Pushgateway plugin")
	return &proto.CleanupResponse{Success: true}, nil
}

// main is the entry point for the plugin binary
func main() {
	impl := NewPrometheusPushgatewayPlugin()

	// Serve the plugin using HashiCorp go-plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: pluginPkg.Handshake,
		Plugins: map[string]plugin.Plugin{
			"observer": &pluginPkg.ObserverPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})

	log.Println("Plugin exited")
}
