package zms

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ZMSConf struct {
	ServerConfig string `yaml:"server_config"`
	Targets      []Target
	TagFilter    Filter   `yaml:"tag_filters"`
	BufferSize   int      `yaml:"buffer_size"`
	Http         HTTPConf `yaml:"http"`
}

type HTTPConf struct {
	ListenPort    int    `yaml:"listen_port"`
	ListenAddress string `yaml:"listen_address"`
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
	if conf.BufferSize == 0 {
		conf.BufferSize = 100
	}

	if len(conf.TagFilter.AcceptedTags) != 0 || len(conf.TagFilter.RejectedTags) != 0 {
		conf.TagFilter.Activate()
	}

	if conf.Http.ListenPort == 0 {
		conf.Http.ListenPort = 2020
	}

	return
}
