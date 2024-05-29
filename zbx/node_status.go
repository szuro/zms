//go:build !debug

package zbx

import (
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const HEADER_LEN = 3
const INITIAL_SYNC = "Cannot perform specified runtime control command during initial configuration cache sync"
const NON_ACTIVE = "Runtime commands can be executed only in active mode"

func GetFailoverDelay(input string) time.Duration {
	re := regexp.MustCompile(`Failover delay: (?P<delay>\d+) seconds`)
	match := re.FindStringSubmatch(input)
	delay, _ := strconv.Atoi(match[1])
	return time.Duration(delay)

}

func ExtractNameAndStatus(input string) (string, string) {
	re := regexp.MustCompile(`\d+\.\s+\S+\s+(\S+)\s+\S+\s+(\S+)`)
	match := re.FindStringSubmatch(input)

	if len(match) > 2 {
		name := strings.TrimSpace(match[1])
		status := strings.TrimSpace(match[2])
		return name, status
	}

	return "", ""
}

func GetHaStatus(config ZabbixConf) (delay time.Duration, nodeIsActive bool) {
	cmd := exec.Command("zabbix_server", "-c", config.configPath, "-R", "ha_status")
	var outString string
	out, err := cmd.Output()

	outString = strings.TrimRight(string(out[:]), "\n")
	if err != nil {
		log.Fatal(err)
	} else if outString == INITIAL_SYNC {
		log.Println("Waiting for initial sync to end...")
		time.Sleep(30 * time.Second)
	}

	lines := strings.Split(outString, "\n")

	if len(lines) == HEADER_LEN {
		log.Println("Node running in standalone mode")
		nodeIsActive = true
	}

	if strings.TrimRight(lines[0], "\n") == NON_ACTIVE {
		log.Println("Node in non-active mode, defaulting to delay=60")
		delay = time.Duration(60)
		return
	} else {
		delay = GetFailoverDelay(lines[0])
	}

	for _, line := range lines[HEADER_LEN:] {
		name, status := ExtractNameAndStatus(line)
		if name == config.NodeName && status == "active" {
			nodeIsActive = true
		}
	}

	return
}
