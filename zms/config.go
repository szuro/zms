package zms

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ZMSConf struct {
	ServerConfig string `yaml:"server_config"`
	Targets      []Target
	TagFilter    Filter `yaml:"tag_filters"`
}

func ParseZMSConfig(path string) (conf ZMSConf) {
	file, err := os.ReadFile(path)
	if err != nil {
		return
	}

	conf = ZMSConf{}
	err = yaml.Unmarshal(file, &conf)
	if conf.ServerConfig == "" {
		conf.ServerConfig = "/etc/zabbix/zabbix_server.conf"
	}

	if len(conf.TagFilter.AcceptedTags) != 0 || len(conf.TagFilter.RejectedTags) != 0 {
		conf.TagFilter.Activate()
	}

	return
}
