package zbx

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	syncerGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "zms_dbsyncers_total",
		Help: "Number of DBSyncer processes",
	})
)
