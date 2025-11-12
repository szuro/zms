package input

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/internal/config"
	"szuro.net/zms/pkg/filter"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

type Subjecter interface {
	AcceptValues()
	Register(observer config.Observer)
	Deregister(observer config.Observer)
	NotifyAll()
	SetFilter(filter filter.Filter)
	Cleanup()
	SetBuffer(size int)
	GetFunnel() chan any
}

type ObserverRegistry map[string]config.Observer

type Subject[T zbxpkg.Export] struct {
	observers        ObserverRegistry
	values           []T
	buffer           int
	Funnel           chan any
	globalFilter     filter.Filter
	bufferSizeGauge  prometheus.Gauge
	bufferUsageGauge prometheus.Gauge
}

func (s *Subject[T]) SetBuffer(size int) {
	s.buffer = size

	var t T
	exportyType := t.GetExportName()
	bufferLabels := prometheus.Labels{"export_type": exportyType}

	s.bufferSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_buffer_size",
		Help:        "Size of internal ZMS buffer",
		ConstLabels: bufferLabels,
	})
	s.bufferSizeGauge.Set(float64(size))

	s.bufferUsageGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_buffer_usage",
		Help:        "Values in internal ZMS buffer",
		ConstLabels: bufferLabels,
	})
	s.bufferUsageGauge.Set(0)
}

func NewSubject[t zbxpkg.Export]() (s Subject[t]) {
	s.observers = make(ObserverRegistry)
	return s
}

func (bs *Subject[T]) Register(observer config.Observer) {
	//nil observer check
	if observer == nil {
		return
	}
	bs.observers[observer.GetName()] = observer
}

func (bs *Subject[T]) Deregister(observer config.Observer) {
	delete(bs.observers, observer.GetName())
}

func (bs *Subject[T]) NotifyAll() {
	var t T
	for _, v := range bs.observers {
		switch any(t).(type) {
		case zbxpkg.History:
			h := any(bs.values).([]zbxpkg.History)
			go v.SaveHistory(h)
		case zbxpkg.Trend:
			t := any(bs.values).([]zbxpkg.Trend)
			go v.SaveTrends(t)
		case zbxpkg.Event:
			e := any(bs.values).([]zbxpkg.Event)
			go v.SaveEvents(e)
		}
	}
}
func (bs *Subject[T]) AcceptValues() {
	for h := range bs.Funnel {
		v := h.(T)
		var accepted bool
		switch h := h.(type) {
		case zbxpkg.History:
			accepted = bs.globalFilter.AcceptHistory(h)
		case zbxpkg.Trend:
			accepted = bs.globalFilter.AcceptTrend(h)
		case zbxpkg.Event:
			accepted = bs.globalFilter.AcceptEvent(h)
		}
		if !accepted {
			continue
		}
		bs.values = append(bs.values, v)
		usage := len(bs.values)

		bs.bufferUsageGauge.Set(float64(usage))

		if usage >= bs.buffer {
			bs.NotifyAll()
			bs.values = nil
			bs.bufferUsageGauge.Set(0)
		}
	}
}

func (bs *Subject[T]) SetFilter(filter filter.Filter) {
	bs.globalFilter = filter
}

func (bs *Subject[T]) Cleanup() {
	for _, observer := range bs.observers {
		observer.Cleanup()
	}
}

func (bs *Subject[T]) GetFunnel() chan any {
	return bs.Funnel
}
