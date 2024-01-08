package zbx

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	conf.ExportTypes = []string{"history", "trends", "events"}

	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("Could not open file: %s", err))
	}
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

	log.Println("Detected configuraion:")
	log.Printf("  ExportDir=%s\n", conf.ExportDir)
	log.Printf("  Syncers=%d\n", conf.DBSyncers)

	return
}
