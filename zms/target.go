package zms

import (
	"fmt"

	"szuro.net/zms/observer"
)

type Target struct {
	Name       string
	Type       string
	Connection string
	Source     []string
}

func (t *Target) ToObserver() (obs observer.Observer) {
	switch t.Type {
	case "print":
		obs = observer.NewPrint(t.Name, t.Connection)
	case "azuretable":
		obs = observer.NewAzureTable(t.Name, t.Connection)
	case "pushgateway":
		obs = observer.NewPushGatewayManager(t.Name, t.Connection)
	case "gcp_cloud_monitor":
		obs = observer.NewCloudMonitor(t.Name, t.Connection)
	default:
		panic(fmt.Sprintf("Target not supported: %s", t.Type))
	}

	return obs
}
