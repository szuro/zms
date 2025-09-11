package zms

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	Version, Commit, BuildDate string
)

var (
	ZmsInfo = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "zms_build_info",
		Help: "ZMS build information",
		ConstLabels: map[string]string{
			"version":    Version,
			"commit":     Commit,
			"build_date": BuildDate,
		},
	})
)
