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
)

const (
	STDOUT      = "stdout"
	STDERR      = "stderr"
	PLUGIN_NAME = "print"
)

// PrintPlugin implements the gRPC observer interface
type PrintPlugin struct {
	proto.UnimplementedObserverServiceServer
	pluginPkg.BaseObserverGRPC
	out io.Writer
}

// NewPrintPlugin creates a new plugin instance
func NewPrintPlugin() *PrintPlugin {
	return &PrintPlugin{
		BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
	}
}

// Initialize configures the plugin with settings from main application
func (p *PrintPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
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

	p.Logger.Info("Print plugin initialized",
		"connection", req.Connection,
		"name", req.Name)

	return &proto.InitializeResponse{Success: true}, nil
}

// SaveHistory processes history data
func (p *PrintPlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
	// Filter history entries
	history := p.FilterHistory(req.History)

	processedCount := int64(0)
	failedCount := int64(0)

	for _, H := range history {
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Value: %v",
			H.Host.Host, H.Name, H.Clock, H.Value)

		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			failedCount++
		} else {
			processedCount++
		}
	}

	return &proto.SaveResponse{
		Success:          failedCount == 0,
		RecordsProcessed: processedCount,
		RecordsFailed:    failedCount,
	}, nil
}

// SaveTrends processes trend data
func (p *PrintPlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
	// Filter trend entries
	trends := p.FilterTrends(req.Trends)

	processedCount := int64(0)
	failedCount := int64(0)

	for _, T := range trends {
		msg := fmt.Sprintf("Host: %s; Item: %s; Time: %d; Min/Max/Avg: %f/%f/%f",
			T.Host.Host, T.Name, T.Clock, T.Min, T.Max, T.Avg)

		_, err := fmt.Fprintln(p.out, msg)
		if err != nil {
			failedCount++
		} else {
			processedCount++
		}
	}

	return &proto.SaveResponse{
		Success:          failedCount == 0,
		RecordsProcessed: processedCount,
		RecordsFailed:    failedCount,
	}, nil
}

// SaveEvents is not supported by this plugin - returns success with no-op
func (p *PrintPlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// Cleanup releases any resources held by the plugin
func (p *PrintPlugin) Cleanup(ctx context.Context, req *proto.CleanupRequest) (*proto.CleanupResponse, error) {
	p.Logger.Info("Cleaning up Print plugin")
	return &proto.CleanupResponse{Success: true}, nil
}

// main is the entry point for the plugin binary
func main() {
	impl := NewPrintPlugin()

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
