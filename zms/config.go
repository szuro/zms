package zms

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
	"szuro.net/zms/zms/filter"
)

type ZMSConf struct {
	ServerConfig string `yaml:"server_config"`
	Targets      []Target
	TagFilter    filter.Filter `yaml:"tag_filters"`
	BufferSize   int           `yaml:"buffer_size"`
	Http         HTTPConf      `yaml:"http"`
	LogLevel     string        `yaml:"log_level"`
	slogLevel    slog.Level    `yaml:"omitempty"`
}

func (zc *ZMSConf) setLogLevel() {
	switch zc.LogLevel {
	case "DEBUG":
		zc.slogLevel = slog.LevelDebug
	case "INFO":
		zc.slogLevel = slog.LevelInfo
	case "WARN":
		zc.slogLevel = slog.LevelWarn
	case "ERROR":
		zc.slogLevel = slog.LevelError
	default:
		zc.slogLevel = slog.LevelInfo
	}
}

func (zc *ZMSConf) GetLogLevel() slog.Level {
	return zc.slogLevel
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
	if err != nil {
		panic("Cannot parse ZMS config!")
	}

	conf.setLogLevel()
	if conf.ServerConfig == "" {
		conf.ServerConfig = "/etc/zabbix/zabbix_server.conf"
	}
	if conf.BufferSize == 0 {
		conf.BufferSize = 100
	}

	conf.TagFilter.Activate()

	if conf.Http.ListenPort == 0 {
		conf.Http.ListenPort = 2020
	}

	return
}
