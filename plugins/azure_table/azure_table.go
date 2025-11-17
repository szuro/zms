package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/hashicorp/go-plugin"

	pluginPkg "zms.szuro.net/pkg/plugin"
	"zms.szuro.net/pkg/proto"
	zbxpkg "zms.szuro.net/pkg/zbx"
)

const (
	PLUGIN_NAME = "azure_table"
)

var info = proto.PluginInfo{
	Name:        PLUGIN_NAME,
	Version:     "1.0.0",
	Author:      "Robert Szulist",
	Description: "Plugin to export Zabbix history and trends to Azure Table Storage",
}

type HistoryEntity struct {
	aztables.Entity
	HostHost, HostName string

	*zbxpkg.History
}

type TrendEntity struct {
	aztables.Entity
	HostHost, HostName string

	*zbxpkg.Trend
}

// AzureTablePlugin implements the gRPC observer interface
type AzureTablePlugin struct {
	proto.UnimplementedObserverServiceServer
	pluginPkg.BaseObserverGRPC
	h *aztables.Client
	t *aztables.Client
}

// NewAzureTablePlugin creates a new plugin instance
func NewAzureTablePlugin() *AzureTablePlugin {
	return &AzureTablePlugin{
		BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
	}
}

// Initialize configures the plugin with settings from main application
func (p *AzureTablePlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	// Call base initialization to handle common setup
	resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
	if err != nil {
		return resp, err
	}

	// Set plugin name for metrics
	p.PluginName = PLUGIN_NAME

	// Initialize Azure Table service client
	service, err := aztables.NewServiceClientWithNoCredential(req.Connection, nil)
	if err != nil {
		p.Logger.Error("Failed to create Azure Table service client", "error", err)
		return &proto.InitializeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create Azure Table service client: %v", err),
		}, err
	}

	p.h = service.NewClient("history")
	p.t = service.NewClient("trends")

	p.Logger.Info("Azure Table plugin initialized",
		"connection", req.Connection,
		"name", req.Name,
	)

	return &proto.InitializeResponse{Success: true, PluginInfo: &info}, nil
}

// SaveHistory processes history data
func (p *AzureTablePlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
	// Filter history entries
	history := p.FilterHistory(req.History)

	processedCount := int64(0)
	failedCount := int64(0)

	for _, H := range history {
		entity := HistoryEntity{
			Entity: aztables.Entity{
				PartitionKey: fmt.Sprint(H.ItemID),
				RowKey:       fmt.Sprintf("%d.%d", H.Clock, H.Ns),
			},
			History:  &H,
			HostHost: H.Host.Host,
			HostName: H.Host.Name,
		}
		entity.Host = nil

		marshalled, err := json.Marshal(entity)
		if err != nil {
			p.Logger.Error("Failed to marshall to Entity", "export", "history", "error", err)
			failedCount++
			continue
		}

		_, err = p.h.AddEntity(ctx, marshalled, nil)
		if err != nil {
			p.Logger.Error("Failed to save entity", "export", "history", "error", err)
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
func (p *AzureTablePlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
	// Filter trend entries
	trends := p.FilterTrends(req.Trends)

	processedCount := int64(0)
	failedCount := int64(0)

	for _, T := range trends {
		entity := TrendEntity{
			Entity: aztables.Entity{
				PartitionKey: fmt.Sprint(T.ItemID),
				RowKey:       fmt.Sprint(T.Clock),
			},
			Trend:    &T,
			HostHost: T.Host.Host,
			HostName: T.Host.Name,
		}
		entity.Host = nil

		marshalled, err := json.Marshal(entity)
		if err != nil {
			p.Logger.Error("Failed to marshall to Entity", "export", "trends", "error", err)
			failedCount++
			continue
		}

		_, err = p.t.AddEntity(ctx, marshalled, nil)
		if err != nil {
			p.Logger.Error("Failed to save entity", "export", "trends", "error", err)
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
func (p *AzureTablePlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
	return &proto.SaveResponse{Success: true, RecordsProcessed: 0}, nil
}

// Cleanup releases any resources held by the plugin
func (p *AzureTablePlugin) Cleanup(ctx context.Context, req *proto.CleanupRequest) (*proto.CleanupResponse, error) {
	p.Logger.Info("Cleaning up Azure Table plugin")
	return &proto.CleanupResponse{Success: true}, nil
}

// main is the entry point for the plugin binary
func main() {
	impl := NewAzureTablePlugin()

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
