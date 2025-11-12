package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/hashicorp/go-plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	pluginPkg "szuro.net/zms/pkg/plugin"
	zbxpkg "szuro.net/zms/pkg/zbx"
	"szuro.net/zms/proto"
)

const (
	PLUGIN_NAME = "postgresql"
)

// PSQLPlugin implements the gRPC observer interface
type PSQLPlugin struct {
	proto.UnimplementedObserverServiceServer
	pluginPkg.BaseObserverGRPC
	dbConn          *sql.DB
	idleConnections prometheus.Gauge
	maxConnections  prometheus.Gauge
	usedConnections prometheus.Gauge
}

// NewPSQLPlugin creates a new plugin instance
func NewPSQLPlugin() *PSQLPlugin {
	return &PSQLPlugin{
		BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
	}
}

// Initialize configures the plugin with settings from main application
func (p *PSQLPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	// Call base initialization to handle common setup
	resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
	if err != nil {
		return resp, err
	}

	// Set plugin name for metrics
	p.PluginName = PLUGIN_NAME

	// Open database connection
	db, err := sql.Open("postgres", req.Connection)
	if err != nil {
		p.Logger.Error("Failed to open connection", "error", err)
		return &proto.InitializeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to open connection: %v", err),
		}, err
	}

	if err := db.Ping(); err != nil {
		p.Logger.Error("Failed to ping database", "error", err)
		db.Close()
		return &proto.InitializeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to ping database: %v", err),
		}, err
	}

	// Apply connection pool options
	for opt, val := range req.Options {
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

	// Initialize connection metrics
	p.idleConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": p.Name, "plugin_name": PLUGIN_NAME, "conn": "idle"},
	})
	p.maxConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": p.Name, "plugin_name": PLUGIN_NAME, "conn": "max"},
	})
	p.usedConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_psql_connection_stats",
		Help:        "Connection stats related to PostgreSQL database",
		ConstLabels: prometheus.Labels{"target_name": p.Name, "plugin_name": PLUGIN_NAME, "conn": "used"},
	})

	p.updateStats()

	p.Logger.Info("PostgreSQL plugin initialized",
		"connection", req.Connection,
		"name", req.Name)

	return &proto.InitializeResponse{Success: true}, nil
}

// SaveHistory processes history data
func (p *PSQLPlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
	// Filter history entries
	history := p.FilterHistory(req.History)

	if len(history) == 0 {
		return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
	}

	processedCount, failedCount := p.saveHistoryToDB(history)

	return &proto.SaveResponse{
		Success:          failedCount == 0,
		RecordsProcessed: processedCount,
		RecordsFailed:    failedCount,
	}, nil
}

// SaveTrends is not supported by this plugin - returns success with no-op
func (p *PSQLPlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// SaveEvents is not supported by this plugin - returns success with no-op
func (p *PSQLPlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// Cleanup releases any resources held by the plugin
func (p *PSQLPlugin) Cleanup(ctx context.Context, req *proto.CleanupRequest) (*proto.CleanupResponse, error) {
	p.Logger.Info("Cleaning up PostgreSQL plugin")
	if p.dbConn != nil {
		p.dbConn.Close()
	}
	return &proto.CleanupResponse{Success: true}, nil
}

// saveHistoryToDB saves history entries to PostgreSQL database
func (p *PSQLPlugin) saveHistoryToDB(h []zbxpkg.History) (int64, int64) {
	base := "INSERT INTO performance.messages (tagname, value, quality, timestamp, servertimestamp) VALUES ($1, $2, $3, $4, $5)"
	historyLen := int64(len(h))

	p.updateStats()

	txn, err := p.dbConn.Begin()
	if err != nil {
		p.Logger.Error("Failed to begin transaction", "error", err)
		return 0, historyLen
	}

	stmt, err := txn.Prepare(base)
	if err != nil {
		txn.Rollback()
		p.Logger.Error("Failed to prepare statement", "error", err)
		return 0, historyLen
	}
	defer stmt.Close()

	processedCount := int64(0)
	for _, H := range h {
		tag := fmt.Sprintf("%s.%s.%s", H.Host.Host, H.Host.Host, H.Name)
		stamp := unixToStamp(H.Clock)
		_, err := stmt.Exec(tag, H.Value, true, stamp, stamp)
		if err != nil {
			txn.Rollback()
			p.Logger.Error("Failed to execute statement", "error", err)
				return 0, historyLen
		}
		processedCount++
	}

	err = txn.Commit()
	if err != nil {
		p.Logger.Error("Failed to commit transaction", "error", err)
		return 0, historyLen
	}

	p.updateStats()
	return processedCount, 0
}

// updateStats updates connection pool statistics
func (p *PSQLPlugin) updateStats() {
	stats := p.dbConn.Stats()
	p.idleConnections.Set(float64(stats.Idle))
	p.usedConnections.Set(float64(stats.InUse))
	p.maxConnections.Set(float64(stats.MaxOpenConnections))
}

// unixToStamp converts Unix timestamp to PostgreSQL timestamp format
func unixToStamp(unix int) string {
	tm := time.Unix(int64(unix), 0)
	return tm.Format("2006-01-02 15:04:05")
}

// main is the entry point for the plugin binary
func main() {
	impl := NewPSQLPlugin()

	// Serve the plugin using HashiCorp go-plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: pluginPkg.Handshake,
		Plugins: map[string]plugin.Plugin{
			"observer": &pluginPkg.ObserverPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})

	log.Println("Plugin exited")
}
