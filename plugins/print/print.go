package main

import (
	"fmt"
	"io"
	"os"

	zbxpkg "szuro.net/zms/pkg/zbx"

	"szuro.net/zms/pkg/plugin"
)

const (
	STDOUT      = "stdout"
	STDERR      = "stderr"
	PLUGIN_NAME = "print"
)

var PluginInfo = plugin.PluginInfo{
	Name:        PLUGIN_NAME,
	Version:     "1.0.0",
	Description: "Writes Zabbix exports to stdout/stderr",
	Author:      "ZMS Example",
}

type Print struct {
	plugin.BaseObserverImpl
	out io.Writer
}

func NewObserver() plugin.Observer {
	return &Print{
		// BaseObserverImpl: *plugin.NewBaseObserver("", PLUGIN_NAME),
	}
}

func (p *Print) Initialize(connection string, options map[string]string) error {
	switch connection {
	case STDERR:
		p.out = os.Stderr
	default:
		p.out = os.Stdout
	}
	return nil
}

func (p *Print) SaveHistory(h []zbxpkg.History) bool {
	for _, H := range h {
		if !p.EvaluateFilter(H.Tags) {
			continue
		}
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %v", H.Host.Host, H.Name, H.Clock, H.Value)
		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			p.Monitor.HistoryValuesFailed.Inc()
		}
		p.Monitor.HistoryValuesSent.Inc()
	}
	return true
}

func (p *Print) SaveTrends(t []zbxpkg.Trend) bool {
	for _, T := range t {
		msg := fmt.Sprintf(
			"Host: %s; Item: %s; Time: %d; Min/Max/Avg: %f/%f/%f",
			T.Host.Host, T.Name, T.Clock, T.Min, T.Max, T.Avg,
		)
		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			p.Monitor.HistoryValuesFailed.Inc()
		}
		p.Monitor.HistoryValuesSent.Inc()
	}
	return true
}

// No SaveEvents. This will cause a panic by default.
