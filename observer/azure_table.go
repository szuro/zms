package observer

import (
	"context"
	"encoding/json"
	"fmt"

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
	for _, H := range h {
		if !az.localFilter.EvaluateFilter(H.Tags) {
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
		marshalled, _ := json.Marshal(entity)
		_, err := az.h.AddEntity(context.TODO(), marshalled, nil)
		az.monitor.historyValuesSent.Inc()
		if err != nil {
			az.monitor.historyValuesFailed.Inc()
		}
	}
	return true
}

func (az *AzureTable) SaveTrends(t []zbx.Trend) bool {
	for _, T := range t {
		if !az.localFilter.EvaluateFilter(T.Tags) {
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
		marshalled, _ := json.Marshal(entity)
		_, err := az.t.AddEntity(context.TODO(), marshalled, nil)
		az.monitor.trendsValuesSent.Inc()
		if err != nil {
			az.monitor.trendsValuesFailed.Inc()
		}
	}
	return true
}
