package observer

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"szuro.net/zms/internal/logger"
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

type AzureTable struct {
	baseObserver
	e *aztables.Client
	h *aztables.Client
	t *aztables.Client
}

func NewAzureTable(name, conn string) (client *AzureTable, err error) {
	client = &AzureTable{
		baseObserver: baseObserver{
			name:         name,
			observerType: "azure_table",
		},
	}
	service, err := aztables.NewServiceClientWithNoCredential(conn, nil)
	if err != nil {
		return nil, err
	}
	client.h = service.NewClient("history")
	client.t = service.NewClient("trends")
	// client.e = service.NewClient("events")

	return
}

func (az *AzureTable) SaveHistory(h []zbxpkg.History) bool {
	return genericSave[zbxpkg.History](
		h,
		func(H zbxpkg.History) bool { return az.localFilter.EvaluateFilter(H.Tags) },
		az.historyFunction,
		nil,
		0,
	)
}

func (az *AzureTable) historyFunction(h []zbxpkg.History) (failed []zbxpkg.History, err error) {
	failed = make([]zbxpkg.History, 0, len(h))
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
			logger.Error("Failed to marshall to Entity", slog.String("name", az.name), slog.String("export", "history"), slog.Any("error", err))
			continue
		}
		_, err = az.h.AddEntity(context.TODO(), marshalled, nil)
		az.monitor.historyValuesSent.Inc()
		if err != nil {
			logger.Error("Failed to save entity", slog.String("name", az.name), slog.String("export", "history"), slog.Any("error", err))
			az.monitor.historyValuesFailed.Inc()
			failed = append(failed, H)
		}
	}
	return failed, err
}

func (az *AzureTable) SaveTrends(t []zbxpkg.Trend) bool {
	return genericSave[zbxpkg.Trend](
		t,
		func(T zbxpkg.Trend) bool { return az.localFilter.EvaluateFilter(T.Tags) },
		az.trendFunction,
		nil,
		0,
	)
}

func (az *AzureTable) trendFunction(t []zbxpkg.Trend) (failed []zbxpkg.Trend, err error) {
	failed = make([]zbxpkg.Trend, 0, len(t))
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
			logger.Error("Failed to marshall to Entity", slog.String("name", az.name), slog.String("export", "trends"), slog.Any("error", err))
			continue
		}
		_, err = az.t.AddEntity(context.TODO(), marshalled, nil)
		az.monitor.trendsValuesSent.Inc()
		if err != nil {
			logger.Error("Failed to save entity", slog.String("name", az.name), slog.String("export", "trends"), slog.Any("error", err))
			az.monitor.historyValuesFailed.Inc()
			failed = append(failed, T)
		}
		az.monitor.trendsValuesFailed.Inc()
	}
	return failed, err
}
