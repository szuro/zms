package zbx

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
	//	"path/filepath"
)

var ZbxRegex = regexp.MustCompile("^(StartDBSyncers|ExportDir|ExportType)=(.*)$")

type ZabbixConf struct {
	ExportDir   string
	ExportTypes []string
	DBSyncers   int
}

func ParseZabbixConfig(path string) (conf ZabbixConf, err error) {
	file, _ := os.Open(path)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		configLine := scanner.Text()
		line := ZbxRegex.FindStringSubmatch(configLine)
		if line != nil {
			switch line[1] {
			case "ExportDir":
				conf.ExportDir = line[2]
			case "ExportType":
				conf.ExportTypes = strings.Split(line[2], ",")
			case "StartDBSyncers":
				conf.DBSyncers, _ = strconv.Atoi(line[2])
			}
		}
	}

	return
}
