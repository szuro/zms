//go:build !debug

package zbx

import (
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"szuro.net/zms/internal/logger"
)

const (
	HEADER_LEN    = 3
	INITIAL_SYNC  = "Cannot perform specified runtime control command during initial configuration cache sync"
	NON_ACTIVE    = "Runtime commands can be executed only in active mode"
	DEFAULT_DELAY = 60 * time.Second
)

// Returns failover delay in seconds
func GetFailoverDelay(input string) time.Duration {
	re := regexp.MustCompile(`Failover delay: (?P<delay>\d+) seconds`)
	match := re.FindStringSubmatch(input)
	delay, _ := strconv.Atoi(match[1])
	return time.Duration(delay) * time.Second

}

// Extracts NodeName and node status from a single line of ha_status output
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

// Returns failover delay in seconds and if the node is active based on zabbix_server command
func GetHaStatus(config ZabbixConf) (delay time.Duration, nodeIsActive bool) {
	cmd := exec.Command("zabbix_server", "-c", config.configPath, "-R", "ha_status")
	var outString string
	out, err := cmd.Output()

	outString = strings.TrimRight(string(out[:]), "\n")
	if err != nil {
		logger.Error("Failed to get HA status", slog.Any("error", err))
	} else if outString == INITIAL_SYNC {
		logger.Info("Waiting for initial sync to end...")
		time.Sleep(DEFAULT_DELAY)
	}

	lines := strings.Split(outString, "\n")

	if len(lines) == HEADER_LEN {
		logger.Info("Node running in standalone mode")
		nodeIsActive = true
	}

	if strings.TrimRight(lines[0], "\n") == NON_ACTIVE {
		var d time.Duration = DEFAULT_DELAY
		logger.Info("Node in non-active mode, waiting", slog.Duration("delay", d))
		delay = time.Duration(d)
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
