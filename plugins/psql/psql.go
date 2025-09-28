package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"szuro.net/zms/internal/logger"
	"szuro.net/zms/pkg/plugin"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

const (
	PLUGIN_NAME = "postgresql"
)

var PluginInfo = plugin.PluginInfo{
	Name:        PLUGIN_NAME,
	Version:     "1.0.0",
	Description: "Stores Zabbix exports in PostgreSQL database",
	Author:      "ZMS",
}

type PSQL struct {
	plugin.BaseObserverImpl
	dbConn          *sql.DB
	idleConnections prometheus.Gauge
	maxConnections  prometheus.Gauge
	usedConnections prometheus.Gauge
}

func NewObserver() plugin.Observer {
	return &PSQL{}
}

func (p *PSQL) Initialize(connection string, options map[string]string) error {
	db, err := sql.Open("postgres", connection)
	if err != nil {
		logger.Error("Failed to open connection", slog.String("name", p.GetName()), slog.Any("error", err))
		return err
	}
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", slog.String("name", p.GetName()), slog.Any("error", err))
		db.Close()
		return err
	}

	for opt, val := range options {
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

	p.idleConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": p.GetName(), "plugin_name": PLUGIN_NAME, "conn": "idle"},
	})
	p.maxConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": p.GetName(), "plugin_name": PLUGIN_NAME, "conn": "max"},
	})
	p.usedConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": p.GetName(), "plugin_name": PLUGIN_NAME, "conn": "used"},
	})

	p.updateStats()

	return nil
}

func (p *PSQL) GetType() string {
	return PLUGIN_NAME
}

func (p *PSQL) Cleanup() {
	if p.dbConn != nil {
		p.dbConn.Close()
	}
	p.BaseObserverImpl.Cleanup()
}

func unixToStamp(unix int) (stamp string) {
	tm := time.Unix(int64(unix), 0)
	return tm.Format("2006-01-02 15:04:05")
}

func (p *PSQL) SaveHistory(h []zbxpkg.History) bool {
	if len(h) == 0 {
		return true
	}

	// Filter history entries
	filtered := make([]zbxpkg.History, 0, len(h))
	for _, history := range h {
		if p.EvaluateFilter(history.Tags) {
			filtered = append(filtered, history)
		}
	}

	if len(filtered) == 0 {
		return true
	}

	return p.saveHistoryToDB(filtered)
}

func (p *PSQL) saveHistoryToDB(h []zbxpkg.History) bool {
	base := "INSERT INTO performance.messages (tagname, value, quality, timestamp, servertimestamp) VALUES ($1, $2, $3, $4, $5)"
	historyLen := float64(len(h))

	p.updateStats()

	txn, err := p.dbConn.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction", slog.String("name", p.GetName()), slog.Any("error", err))
		p.Monitor.HistoryValuesFailed.Add(historyLen)
		return false
	}
	stmt, err := txn.Prepare(base)
	if err != nil {
		txn.Rollback()
		logger.Error("Failed to prepare statement", slog.String("name", p.GetName()), slog.Any("error", err))
		p.Monitor.HistoryValuesFailed.Add(historyLen)
		return false
	}
	defer stmt.Close()

	for _, H := range h {
		tag := fmt.Sprintf("%s.%s.%s", H.Host.Host, H.Host.Host, H.Name)
		stamp := unixToStamp(H.Clock)
		_, err := stmt.Exec(tag, H.Value, true, stamp, stamp)
		if err != nil {
			txn.Rollback()
			logger.Error("Failed to execute statement", slog.String("name", p.GetName()), slog.Any("error", err))
			p.Monitor.HistoryValuesFailed.Add(historyLen)
			return false
		}
		p.Monitor.HistoryValuesSent.Inc()
	}

	err = txn.Commit()
	if err != nil {
		logger.Error("Failed to commit transaction", slog.String("name", p.GetName()), slog.Any("error", err))
		p.Monitor.HistoryValuesFailed.Add(historyLen)
		return false
	}

	p.updateStats()
	return true
}

func (p *PSQL) updateStats() {
	stats := p.dbConn.Stats()
	p.idleConnections.Set(float64(stats.Idle))
	p.usedConnections.Set(float64(stats.InUse))
	p.maxConnections.Set(float64(stats.MaxOpenConnections))
}
