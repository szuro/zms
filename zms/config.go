package zms

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
	"szuro.net/zms/zms/filter"
)

const FILE_MODE = "file"
const HTTP_MODE = "http"

type ZMSConf struct {
	ServerConfig  string `yaml:"server_config"`
	PositionIndex string `yaml:"position_index"`
	Mode          string
	Targets       []Target
	TagFilter     filter.Filter `yaml:"tag_filters"`
	BufferSize    int           `yaml:"buffer_size"`
	Http          HTTPConf      `yaml:"http"`
	LogLevel      string        `yaml:"log_level"`
	slogLevel     slog.Level    `yaml:"omitempty"`
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

	conf.Mode = setMode(conf.Mode)
	conf.BufferSize = setBuffer(conf.BufferSize)
	conf.Http.ListenPort = setPort(conf.Http.ListenPort)
	conf.PositionIndex = setIndex(conf.PositionIndex)

	conf.setLogLevel()
	if conf.ServerConfig == "" {
		conf.ServerConfig = "/etc/zabbix/zabbix_server.conf"
	}

	conf.TagFilter.Activate()

	return
}

func setBuffer(buffer int) int {
	if buffer == 0 {
		buffer = 100
	}
	return buffer
}

func setMode(mode string) string {
	if mode != FILE_MODE && mode != HTTP_MODE {
		mode = FILE_MODE
	}
	return mode
}

func setPort(port int) int {
	if port == 0 {
		port = 2020
	}
	return port
}

func setIndex(path string) string {
	if path == "" {
		path = "/tmp/position.db"
	}
	return path
}
