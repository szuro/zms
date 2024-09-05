package observer

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
	"szuro.net/zms/zbx"
)

const HISTORY_TYPE string = "custom.googleapis.com/zabbix_export/history"
const TREND_TYPE string = "custom.googleapis.com/zabbix_export/trend"

type CloudMonitor struct {
	baseObserver
	client    *monitoring.MetricClient
	ctx       context.Context
	resource  *monitoredres.MonitoredResource
	projectID string
}

func NewCloudMonitor(name, file string) (cm *CloudMonitor, err error) {
	if file != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", file)
	}

	cm = &CloudMonitor{}
	cm.SetName(name)

	cm.ctx = context.Background()
	creds, err := google.FindDefaultCredentials(cm.ctx)
	if err != nil {
		return nil, err
	}

	cm.projectID = "projects/" + creds.ProjectID
	cm.client, err = monitoring.NewMetricClient(cm.ctx, option.WithCredentialsJSON(creds.JSON))

	if err != nil {
		return nil, err
	}

	cm.resource = newResource()
	createHistoryMetric(cm.projectID)
	cm.monitor.initObserverMetrics("gcp_cloud_monitor", name)

	return
}

func (cm *CloudMonitor) sentHistory(metrics map[int]*monitoringpb.TimeSeries) {
	var ts []*monitoringpb.TimeSeries
	for _, value := range metrics {
		ts = append(ts, value)
	}
	l := float64(len(ts))
	cm.monitor.historyValuesSent.Add(l)

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
			cm.monitor.historyValuesFailed.Add(float64(fails))
		}
	} else {
		//assuming all is lost
		cm.monitor.historyValuesFailed.Add(l)
	}
}

func (cm *CloudMonitor) SaveHistory(h []zbx.History) bool {
	metrics := make(map[int]*monitoringpb.TimeSeries, 0)

	for _, hist := range h {
		if !cm.localFilter.EvaluateFilter(hist.Tags) {
			continue
		}
		if hist.Type != zbx.FLOAT && hist.Type != zbx.UNSIGNED {
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

func (cm *CloudMonitor) SaveTrends(t []zbx.Trend) bool {
	panic("not implemented") // TODO: Implement
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

func itemToMetric(item zbx.History) (m *metricpb.Metric) {
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

func itemToPoint(item zbx.History) (p *monitoringpb.Point) {
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

func newTimeSeries(resource *monitoredres.MonitoredResource, item zbx.History) (series *monitoringpb.TimeSeries) {
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
