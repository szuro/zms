package observer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/zbx"
)

type Observer interface {
	Cleanup()
	GetName() string
	SetName(name string)
	SaveHistory(h []zbx.History) bool
	SaveTrends(t []zbx.Trend) bool
}

type baseObserver struct {
	name    string
	monitor obserwerMetrics
}

func (p *baseObserver) GetName() string {
	return p.name
}
func (p *baseObserver) SetName(name string) {
	p.name = name
}

func (p *baseObserver) Cleanup() {

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
		Help:        "The total number of processed lines",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "history"},
	})

	m.historyValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_errors_total",
		Help:        "The total number of processed lines",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "history"},
	})
	m.trendsValuesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_operations_total",
		Help:        "The total number of processed lines",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "trends"},
	})

	m.trendsValuesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "zms_shipping_errors_total",
		Help:        "The total number of processed lines",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "export_type": "trends"},
	})
}
