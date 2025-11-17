package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/hashicorp/go-plugin"
	pluginPkg "zms.szuro.net/pkg/plugin"
	"zms.szuro.net/pkg/proto"
	zbxpkg "zms.szuro.net/pkg/zbx"
)

const (
	STDOUT      = "stdout"
	STDERR      = "stderr"
	PLUGIN_NAME = "print"
)

var info = proto.PluginInfo{
	Name:        PLUGIN_NAME,
	Version:     "1.0.0",
	Author:      "Robert Szulist",
	Description: "Basic print plugin",
}

// LogPrintPlugin implements the gRPC observer interface
type LogPrintPlugin struct {
	proto.UnimplementedObserverServiceServer
	pluginPkg.BaseObserverGRPC
	out io.Writer
}

// NewLogPrintPlugin creates a new plugin instance
func NewLogPrintPlugin() *LogPrintPlugin {
	return &LogPrintPlugin{
		BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
	}
}

// Initialize configures the plugin with settings from main application
func (p *LogPrintPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	// Call base initialization to handle common setup
	resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
	if err != nil {
		return resp, err
	}

	// Set plugin name for metrics
	p.PluginName = PLUGIN_NAME

	// Configure output destination
	switch req.Connection {
	case STDERR:
		p.out = os.Stderr
	default:
		p.out = os.Stdout
	}

	p.Logger.Info("Log print plugin initialized",
		"connection", req.Connection,
		"name", req.Name,
	)

	return &proto.InitializeResponse{Success: true, PluginInfo: &info}, nil
}

// SaveHistory processes history data - only LOG type items
func (p *LogPrintPlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
	// Filter history to only include LOG type and apply tag filters
	history := p.FilterHistory(req.History)

	processedCount := int64(0)
	failedCount := int64(0)

	for _, h := range history {
		// Only process LOG type history items
		if h.Type != zbxpkg.LOG {
			continue
		}

		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %v",
			h.Host.Host, h.Name, h.Clock, h.Value)

		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			failedCount++
			p.Logger.Error("Failed to write log entry", "error", err)
		} else {
			processedCount++
		}
	}

	return &proto.SaveResponse{
		Success:          true,
		RecordsProcessed: processedCount,
		RecordsFailed:    failedCount,
	}, nil
}

// SaveTrends is not supported by this plugin - returns success with no-op
func (p *LogPrintPlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// SaveEvents is not supported by this plugin - returns success with no-op
func (p *LogPrintPlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// Cleanup releases any resources held by the plugin
func (p *LogPrintPlugin) Cleanup(ctx context.Context, req *proto.CleanupRequest) (*proto.CleanupResponse, error) {
	p.Logger.Info("Cleaning up log print plugin")
	return &proto.CleanupResponse{Success: true}, nil
}

// main is the entry point for the plugin binary
func main() {
	impl := NewLogPrintPlugin()

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
