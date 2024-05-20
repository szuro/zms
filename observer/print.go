package observer

import (
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
	for _, H := range h {
		if !p.localFilter.EvaluateFilter(H.Tags) {
			continue
		}
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %s", H.Host.Host, H.Name, H.Clock, H.Value)
		fmt.Fprintln(p.out, msg)
		p.monitor.historyValuesSent.Inc()
	}
	return true
}

func (p *Print) SaveTrends(t []zbx.Trend) bool {
	for _, T := range t {
		if !p.localFilter.EvaluateFilter(T.Tags) {
			continue
		}
		msg := fmt.Sprintf(
			"Host: %s; Item: %s; Time: %d; Min/Max/Avg: %f/%f/%f",
			T.Host.Host, T.Name, T.Clock, T.Min, T.Max, T.Avg,
		)
		fmt.Fprintln(p.out, msg)
		p.monitor.trendsValuesSent.Inc()
	}
	return true
}
