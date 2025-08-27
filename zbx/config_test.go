package zbx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "zabbix_server.conf")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}

func TestParseZabbixConfig_Defaults(t *testing.T) {
	// Empty config file, should use defaults
	path := writeTempConfigFile(t, "")
	conf, err := ParseZabbixConfig(path)
	require.NoError(t, err)
	require.Equal(t, path, conf.configPath)
	require.Equal(t, 4, conf.DBSyncers)
	require.ElementsMatch(t, []string{HISTORY, TREND, EVENT}, conf.ExportTypes)
	require.Empty(t, conf.ExportDir)
	require.Empty(t, conf.NodeName)
}

func TestParseZabbixConfig_AllFields(t *testing.T) {
	content := `
ExportDir=/tmp/export
ExportType=history,trends
StartDBSyncers=8
HANodeName=node1
`
	path := writeTempConfigFile(t, content)
	conf, err := ParseZabbixConfig(path)
	require.NoError(t, err)
	require.Equal(t, "/tmp/export", conf.ExportDir)
	require.ElementsMatch(t, []string{HISTORY, TREND}, conf.ExportTypes)
	require.Equal(t, 8, conf.DBSyncers)
	require.Equal(t, "node1", conf.NodeName)
}

func TestParseZabbixConfig_PartialFields(t *testing.T) {
	content := `
ExportDir=/data
StartDBSyncers=2
`
	path := writeTempConfigFile(t, content)
	conf, err := ParseZabbixConfig(path)
	require.NoError(t, err)
	require.Equal(t, "/data", conf.ExportDir)
	require.Equal(t, 2, conf.DBSyncers)
	require.ElementsMatch(t, []string{HISTORY, TREND, EVENT}, conf.ExportTypes)
	require.Empty(t, conf.NodeName)
}

func TestParseZabbixConfig_InvalidFile(t *testing.T) {
	require.Panics(t, func() {
		ParseZabbixConfig("nonexistent.conf")
	})
}
