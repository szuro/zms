package observer

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"szuro.net/zms/zbx"
)

const (
	STDOUT = "stdout"
	STDERR = "stderr"
)

type Print struct {
	baseObserver
	out io.Writer
}

func NewPrint(name, out string) (p *Print) {
	p = &Print{}
	p.name = name
	if out == STDERR {
		p.out = os.Stderr
	} else {
		p.out = os.Stdout
	}

	p.monitor.initObserverMetrics("print", name)

	return
}

func (p *Print) SaveHistory(h []zbx.History) bool {
	return genericSave[zbx.History](
		h,
		func(H zbx.History) bool { return p.localFilter.EvaluateFilter(H.Tags) },
		p.historyFunction,
		p.buffer,
		p.offlineBufferTTL,
		func(h zbx.History) []byte {
			return []byte("history_" + fmt.Sprint(h.ItemID) + ":" + fmt.Sprint(h.Clock) + ":" + fmt.Sprint(h.Ns))
		},
		func(val []byte) (zbx.History, error) {
			var h zbx.History
			dec := gob.NewDecoder(bytes.NewReader(val))
			err := dec.Decode(&h)
			return h, err
		},
	)
}

func (p *Print) historyFunction(h []zbx.History) (failed []zbx.History, err error) {
	failed = make([]zbx.History, 0, len(h))
	for _, H := range h {
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %s", H.Host.Host, H.Name, H.Clock, H.Value)
		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			p.monitor.historyValuesFailed.Inc()
			failed = append(failed, H)
		}
		p.monitor.historyValuesSent.Inc()
	}
	return failed, err
}

func (p *Print) SaveTrends(t []zbx.Trend) bool {
	return genericSave[zbx.Trend](
		t,
		func(T zbx.Trend) bool { return p.localFilter.EvaluateFilter(T.Tags) },
		p.trendFunction,
		p.buffer,
		p.offlineBufferTTL,
		func(t zbx.Trend) []byte {
			return []byte("trends_" + fmt.Sprint(t.ItemID) + ":" + fmt.Sprint(t.Clock))
		},
		func(val []byte) (zbx.Trend, error) {
			var t zbx.Trend
			dec := gob.NewDecoder(bytes.NewReader(val))
			err := dec.Decode(&t)
			return t, err
		},
	)
}

func (p *Print) trendFunction(t []zbx.Trend) (failed []zbx.Trend, err error) {
	failed = make([]zbx.Trend, 0, len(t))
	for _, T := range t {
		msg := fmt.Sprintf(
			"Host: %s; Item: %s; Time: %d; Min/Max/Avg: %f/%f/%f",
			T.Host.Host, T.Name, T.Clock, T.Min, T.Max, T.Avg,
		)
		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			p.monitor.historyValuesFailed.Inc()
			failed = append(failed, T)
		}
		p.monitor.historyValuesSent.Inc()
	}
	return failed, err
}
