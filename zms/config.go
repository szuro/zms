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
	WorkingDir    string        `yaml:"working_dir"`
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

	conf.setMode(conf.Mode)
	conf.setBuffer(conf.BufferSize)
	conf.setPort(conf.Http.ListenPort)
	conf.setIndex(conf.PositionIndex)
	conf.setOfflineBuffers()

	conf.setLogLevel()
	if conf.ServerConfig == "" {
		conf.ServerConfig = "/etc/zabbix/zabbix_server.conf"
	}

	conf.TagFilter.Activate()

	return
}

func (zc *ZMSConf) setBuffer(buffer int) {
	if buffer == 0 {
		buffer = 100
	}
}

func (zc *ZMSConf) setMode(mode string) {
	switch mode {
	case FILE_MODE:
		zc.Mode = FILE_MODE
	case HTTP_MODE:
		zc.Mode = HTTP_MODE
	default:
		zc.Mode = FILE_MODE
	}
}

func (zc *ZMSConf) setPort(port int) {
	if port == 0 {
		port = 2020
	}
	zc.Http.ListenPort = port
}

func (zc *ZMSConf) setIndex(path string) {
	if path == "" {
		path = zc.WorkingDir + "/position.db"
	}
	zc.PositionIndex = path
}

func (zc *ZMSConf) setOfflineBuffers() {
	for _, target := range zc.Targets {
		if target.OfflineBufferTime < 0 {
			target.OfflineBufferTime = 0
		}
	}
}
