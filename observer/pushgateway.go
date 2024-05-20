package observer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	url_parser "net/url"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"szuro.net/zms/zbx"
)

type PushGatewayManager struct {
	baseObserver
	url      string
	gateways sync.Map
}

func NewPushGatewayManager(name, url string) *PushGatewayManager {
	_, err := url_parser.Parse(url)
	if err != nil {
		panic(err)
	}

	pgm := PushGatewayManager{
		url: url,
	}
	pgm.SetName(name)
	pgm.monitor.initObserverMetrics("pushgateway", name)

	return &pgm
}

func (pgm *PushGatewayManager) SaveHistory(h []zbx.History) bool {
	for _, element := range h {
		if !pgm.localFilter.EvaluateFilter(element.Tags) {
			continue
		}
		hostName := element.Host.Host
		pushGateway, exists := pgm.gateways.Load(hostName)
		if !exists {
			pushGateway = NewPushGateway(hostName, pgm.url)
			pgm.gateways.Store(hostName, pushGateway)
		}
		pushGateway.(PushGateway).hc.history = append(pushGateway.(PushGateway).hc.history, element)
	}

	pgm.gateways.Range(func(key, value interface{}) bool {
		err := value.(PushGateway).pusher.Add()
		pgm.monitor.historyValuesSent.Inc()
		if err != nil {
			pgm.monitor.historyValuesFailed.Inc()
			log.Println(err)
		}
		return true
	})

	return true
}

func (pgm *PushGatewayManager) SaveTrends(t []zbx.Trend) bool {
	panic("not implemented")
}

func (hc HistoryCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(hc, ch)
}

func (hc HistoryCollector) Collect(ch chan<- prometheus.Metric) {
	metrics := make(map[int]prometheus.Metric, 0)

	for _, hist := range hc.history {
		if hist.Type != zbx.FLOAT && hist.Type != zbx.UNSIGNED {
			// Log?
			continue
		}

		hash := md5.Sum([]byte(hist.Name))
		metric_name := "zabbix_" + hex.EncodeToString(hash[:])
		value := hist.Value.(float64)

		metric := prometheus.MustNewConstMetric(
			prometheus.NewDesc(metric_name, hist.Name, []string{"item", "itemid"}, prometheus.Labels{"history": "history"}),
			prometheus.GaugeValue,
			value,
			hist.Name, fmt.Sprintf("%d", hist.ItemID),
		)

		// Replaces duplicate entries for same itemid with newest entry
		metrics[hist.ItemID] = metric
	}

	for _, metric := range metrics {
		ch <- metric
	}
}

type PushGateway struct {
	baseObserver
	pusher *push.Pusher
	hc     *HistoryCollector
	// TODO: add trend collector
}

type HistoryCollector struct {
	history []zbx.History
}

func NewPushGateway(hostName, url string) PushGateway {
	job, _ := os.Hostname()
	_, err := url_parser.Parse(url)
	if err != nil {
		panic(err)
	}

	pg := PushGateway{}

	pg.hc = &HistoryCollector{
		history: []zbx.History{},
	}

	pg.pusher = push.New(url, job).Collector(pg.hc).Grouping("host", hostName)

	return pg
}
