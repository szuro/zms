package observer

import (
	"fmt"
	"io"
	"os"

	zbxpkg "szuro.net/zms/pkg/zbx"
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
	p = &Print{
		baseObserver: baseObserver{
			name:         name,
			observerType: "print",
		},
	}
	if out == STDERR {
		p.out = os.Stderr
	} else {
		p.out = os.Stdout
	}

	return
}

func (p *Print) SaveHistory(h []zbxpkg.History) bool {
	return genericSave[zbxpkg.History](
		h,
		func(H zbxpkg.History) bool { return p.localFilter.EvaluateFilter(H.Tags) },
		p.historyFunction,
		p.buffer,
		p.offlineBufferTTL,
	)
}

func (p *Print) historyFunction(h []zbxpkg.History) (failed []zbxpkg.History, err error) {
	failed = make([]zbxpkg.History, 0, len(h))
	for _, H := range h {
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %v", H.Host.Host, H.Name, H.Clock, H.Value)
		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			p.monitor.historyValuesFailed.Inc()
			failed = append(failed, H)
		}
		p.monitor.historyValuesSent.Inc()
	}
	return failed, err
}

func (p *Print) SaveTrends(t []zbxpkg.Trend) bool {
	return genericSave[zbxpkg.Trend](
		t,
		func(T zbxpkg.Trend) bool { return p.localFilter.EvaluateFilter(T.Tags) },
		p.trendFunction,
		p.buffer,
		p.offlineBufferTTL,
	)
}

func (p *Print) trendFunction(t []zbxpkg.Trend) (failed []zbxpkg.Trend, err error) {
	failed = make([]zbxpkg.Trend, 0, len(t))
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
