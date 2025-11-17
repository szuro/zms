package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/m3db/prometheus_remote_client_golang/promremote"
	"github.com/prometheus/prometheus/prompb"

	pluginPkg "zms.szuro.net/pkg/plugin"
	"zms.szuro.net/pkg/proto"
	zbxpkg "zms.szuro.net/pkg/zbx"
)

const (
	PLUGIN_NAME    = "prometheus_remote_write"
	PLUGIN_VERSION = "1.0.0"
)

// PrometheusRemoteWritePlugin implements the gRPC observer interface
type PrometheusRemoteWritePlugin struct {
	proto.UnimplementedObserverServiceServer
	pluginPkg.BaseObserverGRPC
	client promremote.Client
}

// NewPrometheusRemoteWritePlugin creates a new plugin instance
func NewPrometheusRemoteWritePlugin() *PrometheusRemoteWritePlugin {
	return &PrometheusRemoteWritePlugin{
		BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
	}
}

// Initialize configures the plugin with settings from main application
func (p *PrometheusRemoteWritePlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	// Call base initialization to handle common setup
	resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
	if err != nil {
		return resp, err
	}

	// Set plugin name for metrics
	p.PluginName = PLUGIN_NAME

	// Configure Prometheus remote write client
	cfg := promremote.NewConfig(
		promremote.WriteURLOption(req.Connection),
		promremote.UserAgent(fmt.Sprintf("ZMS - %s %s", PLUGIN_NAME, PLUGIN_VERSION)),
	)

	client, err := promremote.NewClient(cfg)
	if err != nil {
		p.Logger.Error("Failed to create Prometheus remote write client", "error", err)
		return &proto.InitializeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create client: %v", err),
		}, err
	}
	p.client = client

	p.Logger.Info("Prometheus Remote Write plugin initialized",
		"connection", req.Connection,
		"name", req.Name)

	return &proto.InitializeResponse{Success: true}, nil
}

// SaveHistory processes history data
func (p *PrometheusRemoteWritePlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
	// Filter and convert history
	history := p.FilterHistory(req.History)

	// Filter out non-numeric values
	numericHistory := make([]zbxpkg.History, 0, len(history))
	for _, h := range history {
		if h.IsNumeric() {
			numericHistory = append(numericHistory, h)
		}
	}

	if len(numericHistory) == 0 {
		return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
	}

	counter := int64(len(numericHistory))
	wr := zabbixHistoryToWriteRequest(numericHistory)

	_, err := p.client.WriteProto(ctx, wr, promremote.WriteOptions{})
	if err != nil {
		p.Logger.Error("Failed to write history to Prometheus", "error", err)
		return &proto.SaveResponse{
			Success:          false,
			RecordsProcessed: 0,
			RecordsFailed:    counter,
			Error:            err.Error(),
		}, nil
	}


	return &proto.SaveResponse{
		Success:          true,
		RecordsProcessed: counter,
		RecordsFailed:    0,
	}, nil
}

// SaveTrends processes trend data
func (p *PrometheusRemoteWritePlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
	// Filter trends
	trends := p.FilterTrends(req.Trends)

	if len(trends) == 0 {
		return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
	}

	counter := int64(len(trends))
	wr := zabbixTrendsToWriteRequest(trends)

	_, err := p.client.WriteProto(ctx, wr, promremote.WriteOptions{})
	if err != nil {
		p.Logger.Error("Failed to write trends to Prometheus", "error", err)
		return &proto.SaveResponse{
			Success:          false,
			RecordsProcessed: 0,
			RecordsFailed:    counter,
			Error:            err.Error(),
		}, nil
	}


	return &proto.SaveResponse{
		Success:          true,
		RecordsProcessed: counter,
		RecordsFailed:    0,
	}, nil
}

// SaveEvents is not supported by this plugin - returns success with no-op
func (p *PrometheusRemoteWritePlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// Cleanup releases any resources held by the plugin
func (p *PrometheusRemoteWritePlugin) Cleanup(ctx context.Context, req *proto.CleanupRequest) (*proto.CleanupResponse, error) {
	p.Logger.Info("Cleaning up Prometheus Remote Write plugin")
	return &proto.CleanupResponse{Success: true}, nil
}

// zabbixHistoryToWriteRequest converts Zabbix history to Prometheus WriteRequest
func zabbixHistoryToWriteRequest(history []zbxpkg.History) *prompb.WriteRequest {
	promTS := make(map[int]prompb.TimeSeries, len(history))

	for _, H := range history {
		sample := prompb.Sample{
			Value:     H.Value.(float64),
			Timestamp: zabbixClock(H.Clock, H.Ns),
		}

		if ts, ok := promTS[H.ItemID]; ok {
			ts.Samples = append(ts.Samples, sample)
			promTS[H.ItemID] = ts
		} else {
			promTS[H.ItemID] = prompb.TimeSeries{
				Labels:  zabbixToLabels("history", H.Host.Host, fmt.Sprint(H.ItemID), H.Name),
				Samples: []prompb.Sample{sample},
			}
		}
	}

	sendTs := make([]prompb.TimeSeries, 0, len(promTS))
	for _, ts := range promTS {
		// Prometheus Remote Write compatible senders MUST send samples for any given series in timestamp order
		slices.SortFunc(ts.Samples, timestampSort)
		sendTs = append(sendTs, ts)
	}

	return &prompb.WriteRequest{
		Timeseries: sendTs,
	}
}

// zabbixTrendsToWriteRequest converts Zabbix trends to Prometheus WriteRequest
func zabbixTrendsToWriteRequest(trends []zbxpkg.Trend) *prompb.WriteRequest {
	promTS := make(map[string]prompb.TimeSeries, len(trends)*4)
	trendSamples := make(map[string]prompb.Sample, 4)

	for _, H := range trends {
		clock := zabbixClock(H.Clock, 0)

		trendSamples[zbxpkg.TREND_AVG] = prompb.Sample{
			Value:     H.Avg,
			Timestamp: clock,
		}
		trendSamples[zbxpkg.TREND_MIN] = prompb.Sample{
			Value:     H.Min,
			Timestamp: clock,
		}
		trendSamples[zbxpkg.TREND_MAX] = prompb.Sample{
			Value:     H.Max,
			Timestamp: clock,
		}
		trendSamples[zbxpkg.TREND_COUNT] = prompb.Sample{
			Value:     float64(H.Count),
			Timestamp: clock,
		}

		for _, v := range []string{zbxpkg.TREND_AVG, zbxpkg.TREND_MIN, zbxpkg.TREND_MAX, zbxpkg.TREND_COUNT} {
			trendIndex := fmt.Sprintf("%d_%s", H.ItemID, v)

			if ts, ok := promTS[trendIndex]; ok {
				ts.Samples = append(ts.Samples, trendSamples[v])
				promTS[trendIndex] = ts
			} else {
				basicLabels := zabbixToLabels("trends", H.Host.Host, fmt.Sprint(H.ItemID), H.Name)
				basicLabels = append(basicLabels, prompb.Label{Name: "trend_type", Value: v})

				promTS[trendIndex] = prompb.TimeSeries{
					Labels:  basicLabels,
					Samples: []prompb.Sample{trendSamples[v]},
				}
			}
		}
	}

	sendTs := make([]prompb.TimeSeries, 0, len(promTS))
	for _, ts := range promTS {
		// Prometheus Remote Write compatible senders MUST send samples for any given series in timestamp order
		slices.SortFunc(ts.Samples, timestampSort)
		sendTs = append(sendTs, ts)
	}

	return &prompb.WriteRequest{
		Timeseries: sendTs,
	}
}

// zabbixToLabels creates Prometheus labels from Zabbix data
func zabbixToLabels(export, host, itemID, itemName string) []prompb.Label {
	return []prompb.Label{
		{Name: "__name__", Value: fmt.Sprintf("zabbix_%s_export", export)},
		{Name: "host", Value: host},
		{Name: "item_id", Value: itemID},
		{Name: "item_name", Value: itemName},
	}
}

// zabbixClock converts Zabbix clock to Prometheus timestamp (milliseconds)
func zabbixClock(clock, ns int) int64 {
	return time.Unix(int64(clock), int64(ns)).UnixMilli()
}

// timestampSort is a comparison function for sorting samples by timestamp
func timestampSort(i, j prompb.Sample) int {
	return int(i.Timestamp - j.Timestamp)
}

// main is the entry point for the plugin binary
func main() {
	impl := NewPrometheusRemoteWritePlugin()

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
