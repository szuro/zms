package plugin

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/internal/logger"
	pluginPkg "szuro.net/zms/pkg/plugin"
)

type PluginRegistry struct {
	plugins map[string]*LoadedPlugin
	mutex   sync.RWMutex
}

type LoadedPlugin struct {
	Info    pluginPkg.PluginInfo
	Factory pluginPkg.PluginFactory
	Path    string
}

var registry = &PluginRegistry{
	plugins: make(map[string]*LoadedPlugin),
}

func GetRegistry() *PluginRegistry {
	return registry
}

func (pr *PluginRegistry) LoadPlugin(pluginPath string) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	logger.Info("Loading plugin", slog.String("path", pluginPath))

	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", pluginPath, err)
	}

	factorySym, err := p.Lookup("NewObserver")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewObserver function: %w", pluginPath, err)
	}

	factory, ok := factorySym.(func() pluginPkg.Observer)
	if !ok {
		return fmt.Errorf("plugin %s NewObserver function has wrong signature", pluginPath)
	}

	var info pluginPkg.PluginInfo
	if infoSym, err := p.Lookup("PluginInfo"); err == nil {
		if pluginInfo, ok := infoSym.(*pluginPkg.PluginInfo); ok {
			info = *pluginInfo
		}
	}

	// If no info provided, generate basic info from path
	if info.Name == "" {
		base := filepath.Base(pluginPath)
		info.Name = strings.TrimSuffix(base, filepath.Ext(base))
		info.Type = "plugin"
		info.Version = "unknown"
	}

	// Store the plugin
	loadedPlugin := &LoadedPlugin{
		Info:    info,
		Factory: factory,
		Path:    pluginPath,
	}

	pr.plugins[info.Name] = loadedPlugin

	logger.Info("Successfully loaded plugin",
		slog.String("name", info.Name),
		slog.String("version", info.Version),
		slog.String("type", info.Type))

	promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_plugin_info",
		Help:        "Information about loaded plugins",
		ConstLabels: prometheus.Labels{"plugin_name": info.Name, "plugin_type": info.Type, "plugin_version": info.Version},
	}).Set(1)

	return nil
}

// LoadPluginsFromDir loads all .so files from the specified directory
func (pr *PluginRegistry) LoadPluginsFromDir(pluginDir string) error {
	logger.Info("Loading plugins from directory", slog.String("dir", pluginDir))

	pluginPaths, err := filepath.Glob(filepath.Join(pluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files in %s: %w", pluginDir, err)
	}

	var loadErrors []string
	for _, pluginPath := range pluginPaths {
		if err := pr.LoadPlugin(pluginPath); err != nil {
			logger.Error("Failed to load plugin", slog.String("path", pluginPath), slog.Any("error", err))
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", pluginPath, err))
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("failed to load some plugins: %s", strings.Join(loadErrors, "; "))
	}
	logger.Info("Successfully loaded plugins", slog.Int("count", len(pr.plugins)))
	return nil
}

// GetPlugin returns a plugin by name
func (pr *PluginRegistry) GetPlugin(name string) (*LoadedPlugin, bool) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	plugin, exists := pr.plugins[name]
	return plugin, exists
}

// CreateObserver creates a new observer instance from the specified plugin
func (pr *PluginRegistry) CreateObserver(pluginName string) (pluginPkg.Observer, error) {
	plugin, exists := pr.GetPlugin(pluginName)
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	observer := plugin.Factory()
	if observer == nil {
		return nil, fmt.Errorf("plugin %s factory returned nil observer", pluginName)
	}

	return observer, nil
}

// ListPlugins returns information about all loaded plugins
func (pr *PluginRegistry) ListPlugins() []pluginPkg.PluginInfo {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	infos := make([]pluginPkg.PluginInfo, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		infos = append(infos, plugin.Info)
	}

	return infos
}

// IsPluginType returns true if the target type represents a plugin
func IsPluginType(targetType string) bool {
	return strings.HasPrefix(targetType, "plugin:")
}

// ExtractPluginName extracts the plugin name from a target type like "plugin:myplugin"
func ExtractPluginName(targetType string) string {
	if !IsPluginType(targetType) {
		return ""
	}
	return strings.TrimPrefix(targetType, "plugin:")
}
