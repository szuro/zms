package zbx

import (
	"bufio"
	"fmt"
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

	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("Could not open file: %s", err))
	}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		configLine := scanner.Text()
		line := ZbxRegex.FindStringSubmatch(configLine)
		option := line[1]
		value := line[2]
		if line != nil {
			switch option {
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

	return
}
