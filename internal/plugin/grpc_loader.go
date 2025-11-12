package plugin

import (
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"szuro.net/zms/internal/logger"
	pluginPkg "szuro.net/zms/pkg/plugin"
	"szuro.net/zms/proto"
)

// GRPCPluginRegistry manages gRPC-based observer plugins.
type GRPCPluginRegistry struct {
	plugins map[string]*GRPCLoadedPlugin
	mutex   sync.RWMutex
}

// GRPCLoadedPlugin represents a loaded gRPC plugin with its client.
type GRPCLoadedPlugin struct {
	Name   string
	Path   string
	Client *plugin.Client
}

var grpcRegistry = &GRPCPluginRegistry{
	plugins: make(map[string]*GRPCLoadedPlugin),
}

// GetGRPCRegistry returns the global gRPC plugin registry.
func GetGRPCRegistry() *GRPCPluginRegistry {
	return grpcRegistry
}

// LoadPlugin loads a gRPC plugin from the specified path.
// The plugin executable must be a standalone binary that uses HashiCorp go-plugin.
func (pr *GRPCPluginRegistry) LoadPlugin(pluginPath string) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	logger.Info("Loading gRPC plugin", slog.String("path", pluginPath))

	// Extract plugin name from path
	pluginName := filepath.Base(pluginPath)
	pluginName = strings.TrimSuffix(pluginName, filepath.Ext(pluginName))

	// Check if already loaded
	if _, exists := pr.plugins[pluginName]; exists {
		logger.Info("Plugin already loaded", slog.String("name", pluginName))
		return nil
	}

	// Create plugin client configuration
	clientConfig := &plugin.ClientConfig{
		HandshakeConfig: pluginPkg.Handshake,
		Plugins: map[string]plugin.Plugin{
			"observer": &pluginPkg.ObserverPlugin{},
		},
		Cmd:              exec.Command(pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           logger.NewHCLogAdapter(),
	}

	// Create the client
	client := plugin.NewClient(clientConfig)

	// Store the loaded plugin
	loadedPlugin := &GRPCLoadedPlugin{
		Name:   pluginName,
		Path:   pluginPath,
		Client: client,
	}

	pr.plugins[pluginName] = loadedPlugin

	logger.Info("Successfully loaded gRPC plugin",
		slog.String("name", pluginName),
		slog.String("path", pluginPath))

	// Register plugin metrics
	promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "zms_grpc_plugin_info",
		Help:        "Information about loaded gRPC plugins",
		ConstLabels: prometheus.Labels{"plugin_name": pluginName, "plugin_type": "grpc"},
	}).Set(1)

	return nil
}

// LoadPluginsFromDir loads all plugin executables from the specified directory.
// This looks for executable files (no specific extension) in the directory.
func (pr *GRPCPluginRegistry) LoadPluginsFromDir(pluginDir string) error {
	logger.Info("Loading gRPC plugins from directory", slog.String("dir", pluginDir))

	// Look for all files in the directory
	matches, err := filepath.Glob(filepath.Join(pluginDir, "/*"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files in %s: %w", pluginDir, err)
	}

	var loadErrors []string
	loadedCount := 0

	for _, pluginPath := range matches {
		// Check if file is executable
		info, err := exec.LookPath(pluginPath)
		if err != nil || info == "" {
			// Not an executable, skip it
			continue
		}

		if err := pr.LoadPlugin(pluginPath); err != nil {
			logger.Error("Failed to load gRPC plugin", slog.String("path", pluginPath), slog.Any("error", err))
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", pluginPath, err))
		} else {
			loadedCount++
		}
	}

	if len(loadErrors) > 0 {
		logger.Warn("Failed to load some gRPC plugins", slog.String("errors", strings.Join(loadErrors, "; ")))
	}

	logger.Info("Loaded gRPC plugins from directory", slog.Int("count", loadedCount))
	return nil
}

// GetPlugin returns a loaded plugin by name.
func (pr *GRPCPluginRegistry) GetPlugin(name string) (*GRPCLoadedPlugin, bool) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	plugin, exists := pr.plugins[name]
	return plugin, exists
}

// CreateObserver creates a new observer instance from the specified gRPC plugin.
// This connects to the plugin process and returns a gRPC client that implements
// the observer interface.
func (pr *GRPCPluginRegistry) CreateObserver(pluginName string) (proto.ObserverServiceClient, *plugin.Client, error) {
	loadedPlugin, exists := pr.GetPlugin(pluginName)
	if !exists {
		return nil, nil, fmt.Errorf("gRPC plugin %s not found", pluginName)
	}

	// Connect to the plugin
	rpcClient, err := loadedPlugin.Client.Client()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to plugin %s: %w", pluginName, err)
	}

	// Request the plugin interface
	raw, err := rpcClient.Dispense("observer")
	if err != nil {
		loadedPlugin.Client.Kill()
		return nil, nil, fmt.Errorf("failed to dispense observer from plugin %s: %w", pluginName, err)
	}

	// Cast to the observer client
	observerClient, ok := raw.(proto.ObserverServiceClient)
	if !ok {
		loadedPlugin.Client.Kill()
		return nil, nil, fmt.Errorf("plugin %s did not return a valid observer client", pluginName)
	}

	return observerClient, loadedPlugin.Client, nil
}

// CleanupAll shuts down all loaded plugins.
func (pr *GRPCPluginRegistry) CleanupAll() {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	logger.Info("Cleaning up all gRPC plugins")

	for name, plugin := range pr.plugins {
		logger.Info("Killing gRPC plugin", slog.String("name", name))
		plugin.Client.Kill()
	}

	pr.plugins = make(map[string]*GRPCLoadedPlugin)
}

// ListPlugins returns information about all loaded plugins
func (pr *GRPCPluginRegistry) ListPlugins() []pluginPkg.PluginInfo {

	infos := make([]pluginPkg.PluginInfo, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		infos = append(infos, pluginPkg.PluginInfo{
			Name:    plugin.Name,
			Version: "unknown", // Version info can be extended later
		})
	}

	return infos
}
