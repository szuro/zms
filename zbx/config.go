package zbx

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"log/slog"
	//	"path/filepath"
)

var ZbxRegex = regexp.MustCompile("^(StartDBSyncers|ExportDir|ExportType|HANodeName)=(.*)$")

type ZabbixConf struct {
	configPath  string
	ExportDir   string
	ExportTypes []string
	DBSyncers   int
	NodeName    string
}

func ParseZabbixConfig(path string) (conf ZabbixConf, err error) {
	conf.configPath = path
	// zabbix_server.conf defaults
	conf.DBSyncers = 4
	conf.ExportTypes = []string{HISTORY, TREND, EVENT}
	slog.Info("Reading config", slog.Any("path", conf.configPath))

	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("Could not open file: %s", err))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		configLine := scanner.Text()
		line := ZbxRegex.FindStringSubmatch(configLine)
		if line != nil {
			parameter := line[1]
			value := line[2]
			switch parameter {
			case "ExportDir":
				conf.ExportDir = value
			case "ExportType":
				conf.ExportTypes = strings.Split(value, ",")
			case "StartDBSyncers":
				conf.DBSyncers, _ = strconv.Atoi(value)
			case "HANodeName":
				conf.NodeName = value
			}
		}
	}

	slog.Info(
		"Detected config",
		slog.Any("NodeName", conf.NodeName),
		slog.Any("ExportDir", conf.ExportDir),
		slog.Any("Syncers", conf.DBSyncers),
		slog.Any(strings.Join(conf.ExportTypes, ","), conf.ExportTypes),
	)
	syncerGauge.Set(float64(conf.DBSyncers))

	return
}
