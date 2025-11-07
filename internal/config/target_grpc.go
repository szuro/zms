package config

import (
	"context"
	"fmt"

	"szuro.net/zms/internal/plugin"
	pluginPkg "szuro.net/zms/pkg/plugin"
	"szuro.net/zms/proto"
)

// GRPCObserver wraps a gRPC plugin observer for use in ZMS.
type GRPCObserver struct {
	client     proto.ObserverServiceClient
	wrapper    *plugin.GRPCObserverWrapper
	pluginName string
	name       string
}

// ToGRPCObserver creates a gRPC observer from the target configuration.
// This initializes the gRPC plugin, sends configuration, and returns a wrapper.
func (t *Target) ToGRPCObserver(config ZMSConf) (*GRPCObserver, error) {
	// Create observer from gRPC plugin
	client, _, err := plugin.GetGRPCRegistry().CreateObserver(t.PluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC plugin observer %s: %w", t.PluginName, err)
	}

	// Create wrapper
	wrapper := plugin.NewGRPCObserverWrapper(client, t.PluginName)

	// Prepare filter config
	var filterConfig *proto.FilterConfig
	if t.RawFilter != nil {
		filterMap, ok := t.RawFilter.(map[string]any)
		if ok {
			filterConfig = &proto.FilterConfig{}
			if accept, ok := filterMap["accept"].([]any); ok {
				filterConfig.Accept = interfaceSliceToStringSlice(accept)
			}
			if reject, ok := filterMap["reject"].([]any); ok {
				filterConfig.Reject = interfaceSliceToStringSlice(reject)
			}
		}
	}

	// Convert export types
	exports := make([]proto.ExportType, 0, len(t.Source))
	for _, exportType := range t.Source {
		exports = append(exports, pluginPkg.StringToExportType(exportType))
	}

	// Initialize the plugin
	initReq := &proto.InitializeRequest{
		Name:       t.Name,
		Connection: t.Connection,
		Options:    t.Options,
		Exports:    exports,
		Filter:     filterConfig,
	}

	resp, err := client.Initialize(context.Background(), initReq)
	if err != nil {
		wrapper.Cleanup()
		return nil, fmt.Errorf("failed to initialize gRPC plugin observer %s: %w", t.PluginName, err)
	}

	if !resp.Success {
		wrapper.Cleanup()
		return nil, fmt.Errorf("plugin initialization failed: %s", resp.Error)
	}

	return &GRPCObserver{
		client:     client,
		wrapper:    wrapper,
		pluginName: t.PluginName,
		name:       t.Name,
	}, nil
}

// GetClient returns the gRPC client for this observer.
func (o *GRPCObserver) GetClient() proto.ObserverServiceClient {
	return o.client
}

// GetWrapper returns the wrapper for this observer.
func (o *GRPCObserver) GetWrapper() *plugin.GRPCObserverWrapper {
	return o.wrapper
}

// GetName returns the configured name of this observer.
func (o *GRPCObserver) GetName() string {
	return o.name
}

// GetPluginName returns the plugin type name.
func (o *GRPCObserver) GetPluginName() string {
	return o.pluginName
}

// Cleanup releases resources for this observer.
func (o *GRPCObserver) Cleanup() {
	if o.wrapper != nil {
		o.wrapper.Cleanup()
	}
}

// interfaceSliceToStringSlice converts []interface{} to []string.
func interfaceSliceToStringSlice(slice []any) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
