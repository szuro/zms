package observer

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"

	"szuro.net/zms/zbx"
)

type PSQL struct {
	baseObserver
	dbConn *sql.DB
}

func NewPSQL(name, connStr string) (p *PSQL, err error) {
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
	p.dbConn = db

	p.monitor.initObserverMetrics("psql", name)

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

	return true
}

func (p *PSQL) SaveTrends(t []zbx.Trend) bool {
	panic("Not implemented")
}
