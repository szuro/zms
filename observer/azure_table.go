package observer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"szuro.net/crapage/zbx"
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

func NewAzureTable(name, conn string) (client *AzureTable) {
	client = &AzureTable{}
	client.name = name
	service, _ := aztables.NewServiceClientWithNoCredential(conn, nil)
	client.h = service.NewClient("history")
	client.t = service.NewClient("trends")
	// client.e = service.NewClient("events")

	return
}

func (az *AzureTable) SaveHistory(h []zbx.History) bool {
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
		marshalled, _ := json.Marshal(entity)
		az.h.AddEntity(context.TODO(), marshalled, nil)
	}
	return true
}

func (az *AzureTable) SaveTrends(t []zbx.Trend) bool {
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
		marshalled, _ := json.Marshal(entity)
		az.t.AddEntity(context.TODO(), marshalled, nil)
	}
	return true
}
