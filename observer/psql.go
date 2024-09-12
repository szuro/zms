package observer

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"

	"szuro.net/zms/zbx"
)

type PSQL struct {
	baseObserver
	dbConn          *sql.DB
	idleConnections prometheus.Gauge
	maxConnections  prometheus.Gauge
	usedConnections prometheus.Gauge
}

func NewPSQL(name, connStr string, opts map[string]string) (p *PSQL, err error) {
	p = &PSQL{}
	p.name = name

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Println(fmt.Errorf("failed to connect: %v", err))
	}
	if err := db.Ping(); err != nil {
		log.Println(fmt.Errorf("failed to connect: %v", err))
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

	observerType := "psql"
	p.monitor.initObserverMetrics(observerType, name)

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

func (p *PSQL) SaveHistory(h []zbx.History) bool {
	base := "INSERT INTO performance.messages (tagname, value, quality, timestamp, servertimestamp) VALUES ($1, $2, $3, $4, $5)"

	acceptedValues := make([]zbx.History, 0)

	for _, H := range h {
		if !p.localFilter.EvaluateFilter(H.Tags) {
			continue
		}

		acceptedValues = append(acceptedValues, H)
	}

	p.updateStats()

	if len(acceptedValues) == 0 {
		return true
	}

	txn, err := p.dbConn.Begin()
	if err != nil {
		fmt.Println(fmt.Errorf("failed to begin transaction: %v", err))
		return false
	}
	stmt, err := txn.Prepare(base)
	if err != nil {
		txn.Rollback()
		fmt.Println(fmt.Errorf("failed to prepare statement: %v", err))
	}
	defer stmt.Close()

	for _, H := range acceptedValues {
		tag := fmt.Sprintf("%s.%s.%s", H.Host.Host, H.Host.Host, H.Name)
		stamp := unixToStamp(H.Clock)
		_, err := stmt.Exec(tag, H.Value, true, stamp, stamp)
		if err != nil {
			txn.Rollback()
			p.monitor.historyValuesFailed.Inc()
		}

		p.monitor.historyValuesSent.Inc()
	}

	err = txn.Commit()
	if err != nil {
		fmt.Println(fmt.Errorf("failed to commit transaction: %v", err))
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
