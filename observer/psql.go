package observer

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"

	"szuro.net/zms/zbx"
)

type PSQL struct {
	baseObserver
	dbConn *sql.DB
}

func NewPSQL(name, connStr string) (p *PSQL) {
	p = &PSQL{}
	p.name = name

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic("Cannot connect to DB")
	}
	p.dbConn = db

	p.monitor.initObserverMetrics("psql", name)

	return
}
func (p *PSQL) Cleanup() {
	p.dbConn.Close()
}

func (p *PSQL) SaveHistory(h []zbx.History) bool {
	base := "INSERT INTO performance.messages (tagname, value, quality, timestamp, servertimestamp) "

	inserts := []string{}

	values := []interface{}{}

	for _, H := range h {
		if !p.localFilter.EvaluateFilter(H.Tags) {
			continue
		}
		inserts = append(inserts, "(?, ?, ?, ?, ?)")

		tag := fmt.Sprintf("%s.%s.%s", H.Host.Host, H.Host.Host, H.Name)
		//timestamp: 2024-05-08 16:28:33.000000
		//H.Value to string H.Value.(string)????
		values = append(values, tag, H.Value, true, H.Clock, H.Clock)
		p.monitor.historyValuesSent.Inc()
	}

	base = base + strings.Join(inserts, ",")

	txn, _ := p.dbConn.Begin()
	stmt, _ := p.dbConn.Prepare(base)
	stmt.Exec(values...)
	stmt.Close()
	txn.Commit()

	return true
}

func (p *PSQL) SaveTrends(t []zbx.Trend) bool {
	panic("Not implemented")
}
