package zms

import (
	"fmt"

	"szuro.net/zms/observer"
	"szuro.net/zms/zms/filter"
)

type Target struct {
	Name              string
	Type              string
	Connection        string
	OfflineBufferTime int64         `yaml:"offline_buffer_time"` // Time in hours to keep offline buffer
	TagFilter         filter.Filter `yaml:"tag_filters"`
	Source            []string
	Options           map[string]string
}

func (t *Target) ToObserver() (obs observer.Observer, err error) {
	switch t.Type {
	case "print":
		obs = observer.NewPrint(t.Name, t.Connection)
	case "azuretable":
		obs, err = observer.NewAzureTable(t.Name, t.Connection)
	case "pushgateway":
		obs, err = observer.NewPushGatewayManager(t.Name, t.Connection)
	case "gcp_cloud_monitor":
		obs, err = observer.NewCloudMonitor(t.Name, t.Connection)
	case "psql":
		obs, err = observer.NewPSQL(t.Name, t.Connection, t.Options)
	default:
		panic(fmt.Sprintf("Target not supported: %s", t.Type))
	}

	filter := t.TagFilter
	filter.Activate()
	obs.SetFilter(filter)

	return obs, err
}
