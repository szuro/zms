package observer

import (
	"fmt"
	"io"
	"os"

	"szuro.net/crapage/zbx"
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
	return
}

func (p *Print) SaveHistory(h []zbx.History) bool {
	for _, H := range h {
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %s", H.Host.Host, H.Name, H.Clock, H.Value)
		fmt.Fprintln(p.out, msg)
	}
	return true
}

func (p *Print) SaveTrends(t []zbx.Trend) bool {
	for _, T := range t {
		msg := fmt.Sprintf(
			"Host: %s; Item: %s; Time: %d; Min/Max/Avg: %f/%f/%f",
			T.Host.Host, T.Name, T.Clock, T.Min, T.Max, T.Avg,
		)
		fmt.Fprintln(p.out, msg)
	}
	return true
}
