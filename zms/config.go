package zms

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ZMSConf struct {
	ServerConfig string `yaml:"server_config"`
	Targets      []Target
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

	return
}
