package zms

import (
	"fmt"

	"szuro.net/zms/observer"
	"szuro.net/zms/zms/filter"
)

type Target struct {
	Name       string
	Type       string
	Connection string
	TagFilter  filter.Filter `yaml:"tag_filters"`
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
	case "psql":
		obs = observer.NewPSQL(t.Name, t.Connection)
	default:
		panic(fmt.Sprintf("Target not supported: %s", t.Type))
	}

	filter := t.TagFilter
	filter.Activate()
	obs.SetFilter(filter)

	return obs
}
