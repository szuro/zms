package main

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"szuro.net/zms/pkg/plugin"
	zbxpkg "szuro.net/zms/pkg/zbx"
)

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

// Plugin metadata - REQUIRED
var PluginInfo = plugin.PluginInfo{
	Name:        "azure_table",
	Version:     "1.0.0",
	Description: "Azure Table Storage observer plugin",
	Author:      "ZMS",
}

type AzureTable struct {
	plugin.BaseObserverImpl
	// e *aztables.Client
	h *aztables.Client
	t *aztables.Client
}

// Factory function - REQUIRED
func NewObserver() plugin.Observer {
	return &AzureTable{}
}

func (client *AzureTable) Initialize(connection string, options map[string]string) error {
	service, err := aztables.NewServiceClientWithNoCredential(connection, nil)
	if err != nil {
		return err
	}
	client.h = service.NewClient("history")
	client.t = service.NewClient("trends")
	// client.e = service.NewClient("events")

	return nil
}

func (az *AzureTable) SaveHistory(h []zbxpkg.History) bool {
	for _, H := range h {
		if !az.EvaluateFilter(H.Tags) {
			continue
		}
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
			slog.Error("Failed to marshall to Entity", slog.Any("export", "history"), slog.Any("error", err))
			az.Monitor.HistoryValuesFailed.Inc()
			continue
		}
		_, err = az.h.AddEntity(context.TODO(), marshalled, nil)
		if err != nil {
			slog.Error("Failed to save entity", slog.Any("export", "history"), slog.Any("error", err))
			az.Monitor.HistoryValuesFailed.Inc()
		} else {
			az.Monitor.HistoryValuesSent.Inc()
		}
	}
	return true
}

func (az *AzureTable) SaveTrends(t []zbxpkg.Trend) bool {
	for _, T := range t {
		if !az.EvaluateFilter(T.Tags) {
			continue
		}
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
			slog.Error("Failed to marshall to Entity", slog.Any("export", "trends"), slog.Any("error", err))
			az.Monitor.TrendsValuesFailed.Inc()
			continue
		}
		_, err = az.t.AddEntity(context.TODO(), marshalled, nil)
		if err != nil {
			slog.Error("Failed to save entity", slog.Any("export", "trends"), slog.Any("error", err))
			az.Monitor.TrendsValuesFailed.Inc()
		} else {
			az.Monitor.TrendsValuesSent.Inc()
		}
	}
	return true
}
