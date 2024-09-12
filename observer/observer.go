package observer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms/filter"
)

type Observer interface {
	Cleanup()
	GetName() string
	SetName(name string)
	SaveHistory(h []zbx.History) bool
	SaveTrends(t []zbx.Trend) bool
	SaveEvents(e []zbx.Event) bool
	SetFilter(filter filter.Filter)
}

type baseObserver struct {
	name        string
	monitor     obserwerMetrics
	localFilter filter.Filter
}

func (p *baseObserver) GetName() string {
	return p.name
}

func (p *baseObserver) SetName(name string) {
	p.name = name
}

func (p *baseObserver) Cleanup() {

}

func (p *baseObserver) SaveTrends(t []zbx.Trend) bool {
	panic("Not implemented")
}

func (p *baseObserver) SaveEvents(e []zbx.Event) bool {
	panic("Not implemented")
}

func (p *baseObserver) SetFilter(filter filter.Filter) {
	p.localFilter = filter
}

type obserwerMetrics struct {
	historyValuesSent   prometheus.Counter
	historyValuesFailed prometheus.Counter
	trendsValuesSent    prometheus.Counter
	trendsValuesFailed  prometheus.Counter
}

func (m *obserwerMetrics) initObserverMetrics(observerType, name string) {
	m.historyValuesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_operations_total",
		Help:        "Total number of history shipping operations",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "history"},
	})

	m.historyValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_errors_total",
		Help:        "Total number of history shipping errors",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "history"},
	})
	m.trendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_operations_total",
		Help:        "Total number of trends shipping operations",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "trends"},
	})

	m.trendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_errors_total",
		Help:        "Total number of trends shipping errors",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "trends"},
	})
}
