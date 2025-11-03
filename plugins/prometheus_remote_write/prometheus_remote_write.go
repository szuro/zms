package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/m3db/prometheus_remote_client_golang/promremote"
	"github.com/prometheus/prometheus/prompb"
	"szuro.net/zms/internal/logger"
	zbxpkg "szuro.net/zms/pkg/zbx"

	"szuro.net/zms/pkg/plugin"
)

const (
	PLUGIN_NAME    = "prometheus_remote_write"
	PLUGIN_VERSION = "1.0.0"
)

var PluginInfo = plugin.PluginInfo{
	Name:        PLUGIN_NAME,
	Version:     PLUGIN_VERSION,
	Description: "Writes Zabbix exports to stdout/stderr",
	Author:      "ZMS Example",
}

type PrometheusRemoteWrite struct {
	plugin.BaseObserverImpl
	client promremote.Client
}

func NewObserver() plugin.Observer {
	return &PrometheusRemoteWrite{}
}

func (prw *PrometheusRemoteWrite) Initialize(connection string, options map[string]string) error {
	//ignore tls option
	//timeout option
	cfg := promremote.NewConfig(
		promremote.WriteURLOption(connection),
		promremote.UserAgent(fmt.Sprintf("ZMS - %s %s", PLUGIN_NAME, PLUGIN_VERSION)),
	)

	client, err := promremote.NewClient(cfg)
	if err != nil {
		// log.Fatal(fmt.Errorf("unable to construct client: %v", err))
	}
	prw.client = client

	return nil
}

func (prw *PrometheusRemoteWrite) SaveHistory(h []zbxpkg.History) bool {
	h = prw.Filter.FilterHistory(h)
	history := make([]zbxpkg.History, len(h))
	for _, H := range h {
		// prw.Monitor.HistoryValuesSent.Inc()
		if H.IsNumeric() {
			// prw.Monitor.HistoryValuesFailed.Inc()
			history = append(history, H)
		}
	}

	counter := float64(len(history))
	wr := zabbixHistoryToWriteRequest(history)
	prw.Monitor.HistoryValuesSent.Add(counter)
	_, err := prw.client.WriteProto(context.TODO(), wr, promremote.WriteOptions{})

	//TODO : only 5xx and 429 errors
	//TODO : senders MUST retry write requests on HTTP 5xx responses and MUST use a backoff algorithm to prevent overwhelming the server
	if err == nil {
		fetchedHistory, _ := prw.Buffer.FetchHistory(int(counter))
		if len(fetchedHistory) > 0 {
			wr = zabbixHistoryToWriteRequest(fetchedHistory)
			_, err := prw.client.WriteProto(context.TODO(), wr, promremote.WriteOptions{})
			if err == nil {
				prw.Buffer.DeleteHistory(fetchedHistory)
			}
		}
		return true
	}

	prw.Monitor.HistoryValuesFailed.Add(counter)
	prw.Buffer.BufferHistory(history)

	return false
}

func (prw *PrometheusRemoteWrite) SaveTrends(t []zbxpkg.Trend) bool {
	t = prw.Filter.FilterTrends(t)

	counter := float64(len(t))
	wr := zabbixTrendsToWriteRequest(t)
	prw.Monitor.TrendsValuesSent.Add(counter)
	_, err := prw.client.WriteProto(context.TODO(), wr, promremote.WriteOptions{})

	//TODO : only 5xx and 429 errors
	//TODO : senders MUST retry write requests on HTTP 5xx responses and MUST use a backoff algorithm to prevent overwhelming the server
	if err == nil {
		fetchedTrends, _ := prw.Buffer.FetchTrends(int(counter))
		if len(fetchedTrends) > 0 {
			wr = zabbixTrendsToWriteRequest(fetchedTrends)
			_, err := prw.client.WriteProto(context.TODO(), wr, promremote.WriteOptions{})
			if err != nil {
				prw.Buffer.DeleteTrends(fetchedTrends)
			}
		}
		return true
	}

	prw.Monitor.HistoryValuesFailed.Add(counter)
	logger.Error("")
	prw.Buffer.BufferTrends(t)

	return false
}

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

	sendTs := make([]prompb.TimeSeries, 0)
	for _, ts := range promTS {
		//Prometheus Remote Write compatible senders MUST send samples for any given series in timestamp order.
		slices.SortFunc(ts.Samples, timestampSort)
		sendTs = append(sendTs, ts)
	}
	return &prompb.WriteRequest{
		Timeseries: sendTs,
	}
}

func zabbixTrendsToWriteRequest(trends []zbxpkg.Trend) *prompb.WriteRequest {
	promTS := make(map[string]prompb.TimeSeries, len(trends))
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
				// basicLabels := append(basicLabels, )
				promTS[trendIndex] = prompb.TimeSeries{
					Labels:  zabbixToLabels("trends", H.Host.Host, fmt.Sprint(H.ItemID), H.Name),
					Samples: []prompb.Sample{trendSamples[v]},
				}
			}
		}
	}

	sendTs := make([]prompb.TimeSeries, 0)
	for _, ts := range promTS {
		//Prometheus Remote Write compatible senders MUST send samples for any given series in timestamp order.
		slices.SortFunc(ts.Samples, timestampSort)
		sendTs = append(sendTs, ts)
	}
	return &prompb.WriteRequest{
		Timeseries: sendTs,
	}
}

func zabbixToLabels(export, host, itemID, itemName string) []prompb.Label {
	return []prompb.Label{
		prompb.Label{Name: "__name__", Value: fmt.Sprintf("zabbix_%d_export", export)},
		prompb.Label{Name: "host", Value: host},
		prompb.Label{Name: "item_id", Value: itemID},
		prompb.Label{Name: "item_name", Value: itemName},
	}
}

func zabbixClock(clock, ns int) int64 {
	return time.Unix(int64(clock), int64(ns)).UnixMilli()
}

func timestampSort(i, j prompb.Sample) int {
	return int(i.Timestamp - j.Timestamp)

}
