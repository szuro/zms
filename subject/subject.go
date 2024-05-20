package subject

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/observer"
	"szuro.net/zms/zbx"
	"szuro.net/zms/zms/filter"
)

type Subjecter interface {
	AcceptValues()
	Register(observer observer.Observer)
	Deregister(observer observer.Observer)
	NotifyAll()
	SetFilter(filter filter.Filter)
	Cleanup()
	SetBuffer(size int)
}

type ObserverRegistry map[string]observer.Observer

type Subject[T zbx.Export] struct {
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

func NewSubject[t zbx.Export]() (s Subject[t]) {
	s.observers = make(ObserverRegistry)
	return s
}

func (bs *Subject[T]) Register(observer observer.Observer) {
	bs.observers[observer.GetName()] = observer
}

func (bs *Subject[T]) Deregister(observer observer.Observer) {
	delete(bs.observers, observer.GetName())
}

func (bs *Subject[T]) NotifyAll() {
	var t T
	for _, v := range bs.observers {
		switch any(t).(type) {
		case zbx.History:
			h := any(bs.values).([]zbx.History)
			go v.SaveHistory(h)
		}
	}
}
func (bs *Subject[T]) AcceptValues() {
	for h := range bs.Funnel {
		v := h.(T)
		if !bs.globalFilter.EvaluateFilter(v.ShowTags()) {
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

func MkSubjects(zabbix zbx.ZabbixConf, bufferSize int) (obs map[string]Subjecter) {
	obs = make(map[string]Subjecter)
	for _, v := range zabbix.ExportTypes {
		switch v {
		case zbx.HISTORY:
			hs := NewSubject[zbx.History]()
			hs.Funnel = zbx.FileReaderGenerator[zbx.History](zabbix)
			obs[zbx.HISTORY] = &hs
		case zbx.TREND:
			ts := NewSubject[zbx.Trend]()
			ts.Funnel = zbx.FileReaderGenerator[zbx.Trend](zabbix)
			obs[zbx.TREND] = &ts
		default:
			fmt.Printf("Not supported export: %s", v)
		}
	}

	for _, subject := range obs {
		subject.SetBuffer(bufferSize)
	}
	return
}
