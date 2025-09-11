//go:build debug

package zbx

import (
	"time"
	"szuro.net/zms/zms/logger"
)

const DEFAULT_DELAY = 60

func GetFailoverDelay(input string) time.Duration {
	return time.Duration(60)

}

func ExtractNameAndStatus(input string) (string, string) {
	return "test", "active"
}

func GetHaStatus(config ZabbixConf) (delay time.Duration, nodeIsActive bool) {
	logger.Debug("Debug server is always active")
	return GetFailoverDelay(""), true
}
