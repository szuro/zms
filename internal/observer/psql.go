package observer

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"

	"szuro.net/zms/internal/logger"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

type PSQL struct {
	baseObserver
	dbConn          *sql.DB
	idleConnections prometheus.Gauge
	maxConnections  prometheus.Gauge
	usedConnections prometheus.Gauge
}

func NewPSQL(name, connStr string, opts map[string]string) (p *PSQL, err error) {
	observerType := "psql"
	p = &PSQL{
		baseObserver: baseObserver{
			name:         name,
			observerType: observerType,
		},
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Error("Failed to open connection", slog.String("name", name), slog.Any("error", err))
	}
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", slog.String("name", name), slog.Any("error", err))
		db.Close()
		return nil, err
	}

	for opt, val := range opts {
		switch opt {
		case "max_conn":
			maxconn, _ := strconv.Atoi(val)
			db.SetMaxOpenConns(maxconn)
		case "max_idle":
			maxconn, _ := strconv.Atoi(val)
			db.SetMaxIdleConns(maxconn)
		case "max_conn_time":
			dur, _ := time.ParseDuration(val)
			db.SetConnMaxLifetime(dur)
		case "max_idle_time":
			dur, _ := time.ParseDuration(val)
			db.SetConnMaxIdleTime(dur)
		}
	}

	p.dbConn = db

	p.idleConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "conn": "idle"},
	})
	p.maxConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "conn": "max"},
	})

	p.usedConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": name, "target_type": observerType, "conn": "used"},
	})

	p.updateStats()

	return
}
func (p *PSQL) Cleanup() {
	p.dbConn.Close()
}

func unixToStamp(unix int) (stamp string) {
	tm := time.Unix(int64(unix), 0)
	return tm.Format("2006-01-02 15:04:05")
}

func (p *PSQL) SaveHistory(h []zbxpkg.History) bool {
	return genericSave[zbxpkg.History](
		h,
		func(H zbxpkg.History) bool { return p.localFilter.EvaluateFilter(H.Tags) },
		p.historyFunction,
		nil,
		0,
	)
}

func (p *PSQL) historyFunction(h []zbxpkg.History) (failed []zbxpkg.History, err error) {
	base := "INSERT INTO performance.messages (tagname, value, quality, timestamp, servertimestamp) VALUES ($1, $2, $3, $4, $5)"
	historyLen := float64(len(h))
	failed = make([]zbxpkg.History, 0, len(h))

	p.updateStats()

	txn, err := p.dbConn.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction", slog.String("name", p.name), slog.Any("error", err))
		p.monitor.historyValuesFailed.Add(historyLen)
		failed = h
		return failed, err
	}
	stmt, err := txn.Prepare(base)
	if err != nil {
		txn.Rollback()
		logger.Error("Failed to prepare statement", slog.String("name", p.name), slog.Any("error", err))
		p.monitor.historyValuesFailed.Add(historyLen)
		failed = h
		return failed, err
	}
	defer stmt.Close()

	for _, H := range h {
		p.monitor.historyValuesSent.Inc()
		tag := fmt.Sprintf("%s.%s.%s", H.Host.Host, H.Host.Host, H.Name)
		stamp := unixToStamp(H.Clock)
		_, err := stmt.Exec(tag, H.Value, true, stamp, stamp)
		if err != nil {
			txn.Rollback()
			logger.Error("Failed to execute statement", slog.String("name", p.name), slog.Any("error", err))
			p.monitor.historyValuesFailed.Add(historyLen)
			failed = h
			return failed, err
		}

	}

	err = txn.Commit()
	if err != nil {
		logger.Error("Failed to commit transaction", slog.String("name", p.name), slog.Any("error", err))
		p.monitor.historyValuesFailed.Add(historyLen)
		failed = h
		return failed, err
	}

	p.updateStats()
	p.monitor.historyValuesFailed.Add(historyLen)
	failed = h
	return failed, err
}

func (p *PSQL) updateStats() {
	stats := p.dbConn.Stats()
	p.idleConnections.Set(float64(stats.Idle))
	p.usedConnections.Set(float64(stats.InUse))
	p.maxConnections.Set(float64(stats.MaxOpenConnections))
}
