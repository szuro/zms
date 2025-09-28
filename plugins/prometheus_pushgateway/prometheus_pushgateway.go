package main

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"

	"szuro.net/zms/pkg/plugin"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

// Plugin metadata - REQUIRED
var PluginInfo = plugin.PluginInfo{
	Name:        "prometheus_pushgateway",
	Version:     "1.0.0",
	Description: "Prometheus Pushgateway observer plugin",
	Author:      "ZMS",
}

type PrometheusPushgateway struct {
	plugin.BaseObserverImpl
	gatewayURL string
	jobName    string
	registry   *prometheus.Registry
}

// Factory function - REQUIRED
func NewObserver() plugin.Observer {
	return &PrometheusPushgateway{}
}

func (p *PrometheusPushgateway) Initialize(connection string, options map[string]string) error {
	p.gatewayURL = connection
	p.jobName = options["job_name"]
	if p.jobName == "" {
		p.jobName = "zms_export"
	}

	p.registry = prometheus.NewRegistry()
	return nil
}

func (p *PrometheusPushgateway) SaveHistory(h []zbxpkg.History) bool {
	for _, H := range h {
		if !p.EvaluateFilter(H.Tags) {
			continue
		}

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
			p.Monitor.HistoryValuesFailed.Inc()
		} else {
			p.Monitor.HistoryValuesSent.Inc()
		}

		// Unregister for next iteration
		p.registry.Unregister(gauge)
	}
	return true
}

func (p *PrometheusPushgateway) SaveTrends(t []zbxpkg.Trend) bool {
	for _, T := range t {
		if !p.EvaluateFilter(T.Tags) {
			continue
		}

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
			p.Monitor.TrendsValuesFailed.Inc()
		} else {
			p.Monitor.TrendsValuesSent.Inc()
		}

		// Unregister for next iteration
		p.registry.Unregister(minGauge)
		p.registry.Unregister(maxGauge)
		p.registry.Unregister(avgGauge)
	}
	return true
}

// SaveEvents is not implemented - will use default panic behavior from BaseObserver
