package config

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
	"zms.szuro.net/pkg/filter"
)

const FILE_MODE = "file"
const HTTP_MODE = "http"

type ZMSConf struct {
	ServerConfig string `yaml:"server_config"`
	Mode         string
	Targets      []Target
	Filter       filter.FilterConfig `yaml:"filter,omitempty"`
	BufferSize   int                 `yaml:"buffer_size"`
	DataDir      string              `yaml:"data_dir"`
	Http         HTTPConf            `yaml:"http"`
	LogLevel     string              `yaml:"log_level"`
	PluginsDir   string              `yaml:"plugins_dir"` // Directory containing plugin .so files
	slogLevel    slog.Level          `yaml:"omitempty"`
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
		panic("Cannot read ZMS config file! Reason: " + err.Error())
	}

	conf = ZMSConf{}
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		panic("Cannot parse ZMS config! Reason: " + err.Error())
	}

	conf.setMode()
	conf.setBuffer()
	conf.setPort()
	conf.setWorkDir()
	conf.setOfflineBuffers()

	conf.setLogLevel()
	conf.setZbxConf()

	return
}

func (zc *ZMSConf) setBuffer() {
	if zc.BufferSize <= 0 {
		zc.BufferSize = 100
	}
}

func (zc *ZMSConf) setMode() {
	switch zc.Mode {
	case FILE_MODE:
		zc.Mode = FILE_MODE
	case HTTP_MODE:
		zc.Mode = HTTP_MODE
	default:
		zc.Mode = FILE_MODE
	}
}

func (zc *ZMSConf) setPort() {
	if zc.Http.ListenPort == 0 {
		zc.Http.ListenPort = 2020
	}
}

func (zc *ZMSConf) setOfflineBuffers() {
	for i, _ := range zc.Targets {
		if zc.Targets[i].OfflineBufferTime < 0 {
			zc.Targets[i].OfflineBufferTime = 0
		}
	}
}

func (zc *ZMSConf) setWorkDir() {
	if zc.DataDir == "" {
		zc.DataDir = "/var/lib/zms/"
	}
}

func (zc *ZMSConf) setZbxConf() {
	if zc.ServerConfig == "" {
		zc.ServerConfig = "/etc/zabbix/zabbix_server.conf"
	}
}
