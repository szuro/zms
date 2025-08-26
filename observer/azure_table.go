package observer

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"szuro.net/zms/zbx"
)

type HistoryEntity struct {
	aztables.Entity
	HostHost, HostName string

	*zbx.History
}

type TrendEntity struct {
	aztables.Entity
	HostHost, HostName string

	*zbx.Trend
}

type AzureTable struct {
	baseObserver
	e *aztables.Client
	h *aztables.Client
	t *aztables.Client
}

func NewAzureTable(name, conn string) (client *AzureTable, err error) {
	client = &AzureTable{}
	client.name = name
	service, err := aztables.NewServiceClientWithNoCredential(conn, nil)
	if err != nil {
		return nil, err
	}
	client.h = service.NewClient("history")
	client.t = service.NewClient("trends")
	// client.e = service.NewClient("events")
	client.monitor.initObserverMetrics("azure_table", name)

	return
}

func (az *AzureTable) SaveHistory(h []zbx.History) bool {
	return genericSave[zbx.History](
		h,
		func(H zbx.History) bool { return az.localFilter.EvaluateFilter(H.Tags) },
		az.historyFunction,
		az.buffer,
		az.offlineBufferTTL,
	)
}

func (az *AzureTable) historyFunction(h []zbx.History) (failed []zbx.History, err error) {
	failed = make([]zbx.History, 0, len(h))
	for _, H := range h {
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
			slog.Error("Failed to marshall to Entity", slog.Any("name", az.name), slog.Any("export", "history"), slog.Any("error", err))
			continue
		}
		_, err = az.h.AddEntity(context.TODO(), marshalled, nil)
		az.monitor.historyValuesSent.Inc()
		if err != nil {
			slog.Error("Failed to save entity", slog.Any("name", az.name), slog.Any("export", "history"), slog.Any("error", err))
			az.monitor.historyValuesFailed.Inc()
			failed = append(failed, H)
		}
	}
	return failed, err
}

func (az *AzureTable) SaveTrends(t []zbx.Trend) bool {
	return genericSave[zbx.Trend](
		t,
		func(T zbx.Trend) bool { return az.localFilter.EvaluateFilter(T.Tags) },
		az.trendFunction,
		az.buffer,
		az.offlineBufferTTL,
	)
}

func (az *AzureTable) trendFunction(t []zbx.Trend) (failed []zbx.Trend, err error) {
	failed = make([]zbx.Trend, 0, len(t))
	for _, T := range t {
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
			slog.Error("Failed to marshall to Entity", slog.Any("name", az.name), slog.Any("export", "trends"), slog.Any("error", err))
			continue
		}
		_, err = az.t.AddEntity(context.TODO(), marshalled, nil)
		az.monitor.trendsValuesSent.Inc()
		if err != nil {
			slog.Error("Failed to save entity", slog.Any("name", az.name), slog.Any("export", "trends"), slog.Any("error", err))
			az.monitor.historyValuesFailed.Inc()
			failed = append(failed, T)
		}
		az.monitor.trendsValuesFailed.Inc()
	}
	return failed, err
}
