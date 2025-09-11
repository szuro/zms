package observer

import (
	"fmt"
	"log/slog"
	url_parser "net/url"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms/logger"
)

type PushGatewayManager struct {
	baseObserver
	url      string
	gateways sync.Map
}

func NewPushGatewayManager(name, url string) (pgm *PushGatewayManager, err error) {
	_, err = url_parser.Parse(url)
	if err != nil {
		logger.Error("Failed to parse URL", slog.String("name", name), slog.Any("error", err))
		return nil, err
	}

	pgm = &PushGatewayManager{
		url: url,
	}
	pgm.SetName(name)
	pgm.monitor.initObserverMetrics("pushgateway", name)

	return
}

func (pgm *PushGatewayManager) SaveHistory(h []zbx.History) bool {
	acceptedValues := make([]zbx.History, 0, len(h))
	for _, element := range h {
		if !pgm.localFilter.EvaluateFilter(element.Tags) {
			continue
		}
		acceptedValues = append(acceptedValues, element)
	}

	for _, element := range acceptedValues {
		hostName := element.Host.Host
		pg, exists := pgm.gateways.Load(hostName)
		if !exists {
			pg = newPushGateway(hostName, pgm.url)
			pgm.gateways.Store(hostName, pg)
		}
		pg.(pushGateway).hc.history = append(pg.(pushGateway).hc.history, element)
	}

	pgm.gateways.Range(func(key, value interface{}) bool {
		pusher := value.(pushGateway).pusher
		err := pusher.Add()
		pgm.monitor.historyValuesSent.Inc()
		if err != nil {
			pgm.monitor.historyValuesFailed.Inc()
			logger.Error("Failed to ship values", slog.String("name", pgm.name), slog.Any("error", err))
		}
		return true
	})

	return true
}

func (hc historyCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(hc, ch)
}

func (hc historyCollector) Collect(ch chan<- prometheus.Metric) {
	metrics := make(map[int]prometheus.Metric, 0)

	for _, hist := range hc.history {
		if !hist.IsNumeric() {
			// Log?
			continue
		}

		metric := prometheus.MustNewConstMetric(
			prometheus.NewDesc("zabbix_push_history", hist.Name, []string{"item", "itemid"}, prometheus.Labels{"history": "history"}),
			prometheus.GaugeValue,
			hist.Value.(float64),
			hist.Name, fmt.Sprintf("%d", hist.ItemID),
		)

		// Replaces duplicate entries for same itemid with newest entry
		metrics[hist.ItemID] = metric
	}

	for _, metric := range metrics {
		ch <- metric
	}
}

type pushGateway struct {
	pusher *push.Pusher
	hc     *historyCollector
	// TODO: add trend collector
}

type historyCollector struct {
	history []zbx.History
}

func newPushGateway(hostName, url string) pushGateway {
	job, _ := os.Hostname()
	_, err := url_parser.Parse(url)
	if err != nil {
		panic(err)
	}

	pg := pushGateway{}

	pg.hc = &historyCollector{
		history: []zbx.History{},
	}

	pg.pusher = push.New(url, job).Collector(pg.hc).Grouping("host", hostName)

	return pg
}
