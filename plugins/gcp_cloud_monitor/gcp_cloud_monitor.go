package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/hashicorp/go-plugin"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/label"
	"google.golang.org/genproto/googleapis/api/metric"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"

	pluginPkg "zms.szuro.net/pkg/plugin"
	"zms.szuro.net/pkg/proto"
	zbxpkg "zms.szuro.net/pkg/zbx"
)

const (
	PLUGIN_NAME  = "gcp_cloud_monitor"
	HISTORY_TYPE = "custom.googleapis.com/zabbix_export/history"
	TREND_TYPE   = "custom.googleapis.com/zabbix_export/trend"
)

// GCPCloudMonitorPlugin implements the gRPC observer interface
type GCPCloudMonitorPlugin struct {
	proto.UnimplementedObserverServiceServer
	pluginPkg.BaseObserverGRPC
	client    *monitoring.MetricClient
	ctx       context.Context
	resource  *monitoredres.MonitoredResource
	projectID string
}

// NewGCPCloudMonitorPlugin creates a new plugin instance
func NewGCPCloudMonitorPlugin() *GCPCloudMonitorPlugin {
	return &GCPCloudMonitorPlugin{
		BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
	}
}

// Initialize configures the plugin with settings from main application
func (p *GCPCloudMonitorPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	// Call base initialization to handle common setup
	resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
	if err != nil {
		return resp, err
	}

	// Set plugin name for metrics
	p.PluginName = PLUGIN_NAME

	// Set credentials file if provided
	if credFile := req.Options["credentials_file"]; credFile != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credFile)
	}

	p.ctx = ctx
	creds, err := google.FindDefaultCredentials(p.ctx)
	if err != nil {
		p.Logger.Error("Failed to find Google Cloud credentials", "error", err)
		return &proto.InitializeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to find credentials: %v", err),
		}, err
	}

	p.projectID = "projects/" + creds.ProjectID
	p.client, err = monitoring.NewMetricClient(p.ctx, option.WithCredentialsJSON(creds.JSON))
	if err != nil {
		p.Logger.Error("Failed to create metric client", "error", err)
		return &proto.InitializeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create metric client: %v", err),
		}, err
	}

	p.resource = newResource()
	createHistoryMetric(p.projectID)

	p.Logger.Info("GCP Cloud Monitor plugin initialized",
		"project", creds.ProjectID,
		"name", req.Name)

	return &proto.InitializeResponse{Success: true}, nil
}

// SaveHistory processes history data
func (p *GCPCloudMonitorPlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
	// Filter history entries
	history := p.FilterHistory(req.History)
	fails := int64(0)
	metrics := make(map[int]*monitoringpb.TimeSeries, 0)

	for _, hist := range history {
		// Only process numeric values
		if hist.Type != zbxpkg.FLOAT && hist.Type != zbxpkg.UNSIGNED {
			continue
		}

		if _, ok := metrics[hist.ItemID]; !ok {
			metrics[hist.ItemID] = newTimeSeries(p.resource, hist)
		} else {
			// Send and clear
			fails += p.sendHistory(metrics)
			metrics = make(map[int]*monitoringpb.TimeSeries, 0)
		}
	}

	// Send leftovers
	if len(metrics) > 0 {
		fails += p.sendHistory(metrics)
	}

	return &proto.SaveResponse{
		Success:          true,
		RecordsProcessed: int64(len(history)),
		RecordsFailed:    fails,
	}, nil
}

// SaveTrends is not supported by this plugin - returns success with no-op
func (p *GCPCloudMonitorPlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// SaveEvents is not supported by this plugin - returns success with no-op
func (p *GCPCloudMonitorPlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// Cleanup releases any resources held by the plugin
func (p *GCPCloudMonitorPlugin) Cleanup(ctx context.Context, req *proto.CleanupRequest) (*proto.CleanupResponse, error) {
	p.Logger.Info("Cleaning up GCP Cloud Monitor plugin")
	if p.client != nil {
		p.client.Close()
	}
	return &proto.CleanupResponse{Success: true}, nil
}

// sendHistory sends time series data to GCP
func (p *GCPCloudMonitorPlugin) sendHistory(metrics map[int]*monitoringpb.TimeSeries) (fails int64) {
	var ts []*monitoringpb.TimeSeries
	for _, value := range metrics {
		ts = append(ts, value)
	}
	l := int64(len(ts))

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       p.projectID,
		TimeSeries: ts,
	}
	err := p.client.CreateTimeSeries(p.ctx, req)

	if aErr, ok := apierror.FromError(err); ok {
		details := aErr.Details()
		if len(details.Unknown) > 0 {
			summary := details.Unknown[0].(*monitoringpb.CreateTimeSeriesSummary)
			fails = int64(summary.TotalPointCount - summary.SuccessPointCount)
		}
	} else if err != nil {
		// Assuming all is lost
		fails = l
	}
	return fails
}

// newResource creates a monitored resource
func newResource() *monitoredres.MonitoredResource {
	host, _ := os.Hostname()
	return &monitoredres.MonitoredResource{
		Type: "generic_task",
		Labels: map[string]string{
			"location":  "global",
			"namespace": "default",
			"job":       "Zabbix Export",
			"task_id":   host,
		},
	}
}

// itemToMetric converts a Zabbix history item to a GCP metric
func itemToMetric(item zbxpkg.History) *metricpb.Metric {
	return &metricpb.Metric{
		Type: HISTORY_TYPE,
		Labels: map[string]string{
			"item":   item.Name,
			"itemid": strconv.Itoa(item.ItemID),
			"host":   item.Host.Host,
		},
	}
}

// itemToPoint converts a Zabbix history item to a GCP point
func itemToPoint(item zbxpkg.History) *monitoringpb.Point {
	stamp := &timestamp.Timestamp{
		Seconds: int64(item.Clock),
	}
	return &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			StartTime: stamp,
			EndTime:   stamp,
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DoubleValue{
				DoubleValue: item.Value.(float64),
			},
		},
	}
}

// newTimeSeries creates a new time series for a Zabbix item
func newTimeSeries(resource *monitoredres.MonitoredResource, item zbxpkg.History) *monitoringpb.TimeSeries {
	return &monitoringpb.TimeSeries{
		Metric:   itemToMetric(item),
		Points:   []*monitoringpb.Point{itemToPoint(item)},
		Resource: resource,
	}
}

// mkStandardLabels creates standard label descriptors
func mkStandardLabels() []*label.LabelDescriptor {
	return []*label.LabelDescriptor{
		{
			Key:         "item",
			ValueType:   label.LabelDescriptor_STRING,
			Description: "Name of a Zabbix item",
		},
		{
			Key:         "itemid",
			ValueType:   label.LabelDescriptor_INT64,
			Description: "itemid of a Zabbix item",
		},
		{
			Key:         "host",
			ValueType:   label.LabelDescriptor_STRING,
			Description: "Host that contains this item",
		},
	}
}

// createHistoryMetric creates the history metric descriptor in GCP
func createHistoryMetric(projectID string) (*metricpb.MetricDescriptor, error) {
	ctx := context.Background()
	c, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	md := &metric.MetricDescriptor{
		Name:        "Zabbix history",
		Type:        HISTORY_TYPE,
		Labels:      mkStandardLabels(),
		MetricKind:  metric.MetricDescriptor_GAUGE,
		ValueType:   metric.MetricDescriptor_DOUBLE,
		Unit:        "",
		Description: "Zabbix item history exported via ZMS",
		DisplayName: "Zabbix history",
	}

	req := &monitoringpb.CreateMetricDescriptorRequest{
		Name:             projectID,
		MetricDescriptor: md,
	}

	m, err := c.CreateMetricDescriptor(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not create custom metric: %w", err)
	}

	return m, nil
}

// main is the entry point for the plugin binary
func main() {
	impl := NewGCPCloudMonitorPlugin()

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
