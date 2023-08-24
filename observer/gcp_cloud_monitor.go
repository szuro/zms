package observer

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"log"
	"os"
	"strconv"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"szuro.net/zms/zbx"
)

type CloudMonitor struct {
	baseObserver
	client    *monitoring.MetricClient
	ctx       context.Context
	resource  *monitoredres.MonitoredResource
	projectID string
}

func NewCloudMonitor(name, file string) (cm *CloudMonitor) {
	if file != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", file)
	}

	cm = &CloudMonitor{}
	cm.SetName(name)

	cm.ctx = context.Background()
	creds, err := google.FindDefaultCredentials(cm.ctx)
	if err != nil {
		// TODO: Handle error.
	}

	cm.projectID = "projects/" + creds.ProjectID
	cm.client, err = monitoring.NewMetricClient(cm.ctx, option.WithCredentialsJSON(creds.JSON))
	cm.resource = newResource()

	if err != nil {
		// TODO: Handle error.
	}

	return
}

func (cm *CloudMonitor) SaveHistory(h []zbx.History) bool {
	metrics := make(map[int]*monitoringpb.TimeSeries, 0)

	for _, hist := range h {
		if hist.Type != zbx.FLOAT && hist.Type != zbx.UNSIGNED {
			continue
		}
		if _, ok := metrics[hist.ItemID]; ok {
			// val.Points = append(val.Points, itemToPoint(hist))
			// timeSerier requires only one data point. Push instead of appending.
			pushTimeSeries(cm, &metrics)
		}
		metrics[hist.ItemID] = newTimeSeries(cm.resource, hist)

	}

	pushTimeSeries(cm, &metrics)

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
	hash := md5.Sum([]byte(item.Name))
	m = &metricpb.Metric{
		Type: "custom.googleapis.com/" + "zabbix_" + hex.EncodeToString(hash[:]),
		Labels: map[string]string{
			"item":    item.Name,
			"itemid":  strconv.Itoa(item.ItemID),
			"host":    item.Host.Host,
			"history": "history",
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

func pushTimeSeries(cm *CloudMonitor, metrics *map[int]*monitoringpb.TimeSeries) {
	var ts []*monitoringpb.TimeSeries

	for _, value := range *metrics {
		ts = append(ts, value)
	}

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       cm.projectID,
		TimeSeries: ts,
	}

	err := cm.client.CreateTimeSeries(cm.ctx, req)
	if err != nil {
		log.Println(err)
	}

	//clear
	*metrics = make(map[int]*monitoringpb.TimeSeries, 0)
}
