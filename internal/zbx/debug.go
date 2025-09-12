//go:build debug

package zbx

import (
	"time"

	"szuro.net/zms/internal/logger"
)

const DEFAULT_DELAY = 60 * time.Second

func GetFailoverDelay(input string) time.Duration {
	return DEFAULT_DELAY
}

func ExtractNameAndStatus(input string) (string, string) {
	return "test", "active"
}

func GetHaStatus(config ZabbixConf) (delay time.Duration, nodeIsActive bool) {
	logger.Debug("Debug server is always active")
	return GetFailoverDelay(""), true
}
