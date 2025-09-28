package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/genproto/googleapis/api/label"
	"google.golang.org/genproto/googleapis/api/metric"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleapis/gax-go/v2/apierror"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/option"
	"szuro.net/zms/pkg/plugin"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

const HISTORY_TYPE string = "custom.googleapis.com/zabbix_export/history"
const TREND_TYPE string = "custom.googleapis.com/zabbix_export/trend"

// Plugin metadata - REQUIRED
var PluginInfo = plugin.PluginInfo{
	Name:        "gcp_cloud_monitor",
	Version:     "1.0.0",
	Description: "Google Cloud Monitoring observer plugin",
	Author:      "ZMS",
}

type CloudMonitor struct {
	plugin.BaseObserverImpl
	client    *monitoring.MetricClient
	ctx       context.Context
	resource  *monitoredres.MonitoredResource
	projectID string
}

// Factory function - REQUIRED
func NewObserver() plugin.Observer {
	return &CloudMonitor{}
}

func (cm *CloudMonitor) Initialize(connection string, options map[string]string) error {
	if credFile := options["credentials_file"]; credFile != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credFile)
	}

	cm.ctx = context.Background()
	creds, err := google.FindDefaultCredentials(cm.ctx)
	if err != nil {
		return err
	}

	cm.projectID = "projects/" + creds.ProjectID
	cm.client, err = monitoring.NewMetricClient(cm.ctx, option.WithCredentialsJSON(creds.JSON))

	if err != nil {
		return err
	}

	cm.resource = newResource()
	createHistoryMetric(cm.projectID)

	return nil
}

func (cm *CloudMonitor) sentHistory(metrics map[int]*monitoringpb.TimeSeries) {
	var ts []*monitoringpb.TimeSeries
	for _, value := range metrics {
		ts = append(ts, value)
	}
	l := float64(len(ts))

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       cm.projectID,
		TimeSeries: ts,
	}
	err := cm.client.CreateTimeSeries(cm.ctx, req)

	if aErr, ok := apierror.FromError(err); ok {
		details := aErr.Details()
		if len(details.Unknown) > 0 {
			summary := details.Unknown[0].(*monitoringpb.CreateTimeSeriesSummary)
			fails := summary.TotalPointCount - summary.SuccessPointCount
			cm.Monitor.HistoryValuesFailed.Add(float64(fails))
			cm.Monitor.HistoryValuesSent.Add(l - float64(fails))
		}
	} else if err != nil {
		//assuming all is lost
		cm.Monitor.HistoryValuesFailed.Add(l)
	} else {
		cm.Monitor.HistoryValuesSent.Add(l)
	}
}

func (cm *CloudMonitor) SaveHistory(h []zbxpkg.History) bool {
	metrics := make(map[int]*monitoringpb.TimeSeries, 0)

	for _, hist := range h {
		if !cm.EvaluateFilter(hist.Tags) {
			continue
		}
		if hist.Type != zbxpkg.FLOAT && hist.Type != zbxpkg.UNSIGNED {
			continue
		}

		if _, ok := metrics[hist.ItemID]; !ok {
			metrics[hist.ItemID] = newTimeSeries(cm.resource, hist)
		} else {
			//sent and clear
			cm.sentHistory(metrics)
			metrics = make(map[int]*monitoringpb.TimeSeries, 0)
		}
	}

	//sent leftovers
	if len(metrics) > 0 {
		cm.sentHistory(metrics)
	}

	return true
}

func (cm *CloudMonitor) Cleanup() {
	cm.client.Close()
}

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

func itemToMetric(item zbxpkg.History) (m *metricpb.Metric) {
	m = &metricpb.Metric{
		Type: HISTORY_TYPE,
		Labels: map[string]string{
			"item":   item.Name,
			"itemid": strconv.Itoa(item.ItemID),
			"host":   item.Host.Host,
		},
	}
	return
}

func itemToPoint(item zbxpkg.History) (p *monitoringpb.Point) {
	stamp := &timestamp.Timestamp{
		Seconds: int64(item.Clock),
	}
	p = &monitoringpb.Point{
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
	return
}

func newTimeSeries(resource *monitoredres.MonitoredResource, item zbxpkg.History) (series *monitoringpb.TimeSeries) {
	series = &monitoringpb.TimeSeries{
		Metric:   itemToMetric(item),
		Points:   []*monitoringpb.Point{itemToPoint(item)},
		Resource: resource,
	}
	return
}

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
		}, {
			Key:         "host",
			ValueType:   label.LabelDescriptor_STRING,
			Description: "Host that contains this item",
		},
	}
}

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
