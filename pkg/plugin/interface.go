// Package plugin provides interfaces and types for creating ZMS observer plugins.
//
// This package defines the plugin system for ZMS (Zabbix Metric Shipper), enabling
// developers to create custom observer implementations that can process Zabbix export
// data and send it to various destinations.
//
// The plugin system supports dynamic loading of shared libraries (.so files) that
// implement the Observer interface. Plugins have access to core ZMS functionality
// through the BaseObserver interface, including filtering, metrics, and buffering.
//
// Creating a Plugin:
//
// 1. Implement the Observer interface
// 2. Embed BaseObserverImpl for core functionality
// 3. Export PluginInfo variable and NewObserver() function
// 4. Compile as shared library: go build -buildmode=plugin
//
// Example plugin structure:
//
//	package main
//
//	import "szuro.net/zms/pkg/plugin"
//
//	var PluginInfo = plugin.PluginInfo{
//	    Name: "my-plugin",
//	    Version: "1.0.0",
//	    Description: "Example plugin",
//	    Author: "Developer",
//	}
//
//	type MyPlugin struct {
//	    plugin.BaseObserverImpl
//	    // Custom fields
//	}
//
//	func NewObserver() plugin.Observer {
//	    return &MyPlugin{}
//	}
//
//	func (p *MyPlugin) Initialize(connection string, options map[string]string) error {
//	    // Plugin initialization logic
//	    return nil
//	}
//
//	func (p *MyPlugin) SaveHistory(h []zbx.History) bool {
//	    // Process history data
//	    return true
//	}
package plugin

import (
	"szuro.net/zms/pkg/filter"
	"szuro.net/zms/pkg/zbx"
)

// Observer defines the interface that all observer plugins must implement.
// This interface provides the contract for processing Zabbix export data and
// integrating with the ZMS plugin system.
//
// Plugins should embed BaseObserverImpl to get default implementations of most
// methods and focus on implementing the core data processing methods and Initialize.
type Observer interface {
	// Core observer lifecycle methods

	// Cleanup releases any resources held by the observer.
	// Called when the observer is being shut down or removed.
	Cleanup()

	// GetName returns the configured name of this observer instance.
	GetName() string

	// SetName sets the name of this observer instance.
	// Used by ZMS to assign the configured target name.
	SetName(name string)

	// InitBuffer initializes the offline buffer for this observer.
	// path: directory path for buffer storage
	// ttl: time-to-live in hours for buffered data
	InitBuffer(path string, ttl int64)

	// Data processing methods - implement these for your plugin logic

	// SaveHistory processes and saves history data.
	// Returns true if processing was successful, false otherwise.
	SaveHistory(h []zbx.History) bool

	// SaveTrends processes and saves trend data.
	// Returns true if processing was successful, false otherwise.
	SaveTrends(t []zbx.Trend) bool

	// SaveEvents processes and saves event data.
	// Returns true if processing was successful, false otherwise.
	SaveEvents(e []zbx.Event) bool

	// Configuration and setup methods

	// SetFilter configures the tag filter for this observer.
	// Used to filter which data should be processed by this observer.
	SetFilter(filter any)

	// PrepareMetrics initializes Prometheus metrics for the specified export types.
	// exports: list of export types this observer will handle ("history", "trends", "events")
	PrepareMetrics(exports []string)

	// Plugin-specific initialization

	// Initialize sets up the plugin with connection details and options.
	// This is called once when the plugin is loaded and configured.
	// connection: connection string specific to the plugin type
	// options: key-value pairs of plugin-specific configuration options
	// Returns error if initialization fails.
	Initialize(connection string, options map[string]string) error
}

// BaseObserver provides access to core ZMS functionality for plugins.
// This interface is implemented by BaseObserverImpl and provides plugins
// with common functionality like filtering, metrics, and buffering without
// requiring them to implement these features from scratch.
//
// Plugins should embed BaseObserverImpl in their struct to automatically
// get an implementation of this interface.
type BaseObserver interface {
	// GetName returns the configured name of this observer instance.
	GetName() string

	// SetName sets the name of this observer instance.
	SetName(name string)

	// InitBuffer initializes the offline buffer for this observer.
	// path: directory path for buffer storage
	// ttl: time-to-live in hours for buffered data
	InitBuffer(path string, ttl int64)

	// SetFilter configures the tag filter for this observer.
	SetFilter(filter filter.Filter)

	// PrepareMetrics initializes Prometheus metrics for the specified export types.
	PrepareMetrics(exports []string)

	// Cleanup releases resources held by the base observer.
	Cleanup()
}

// PluginFactory is the function signature that plugins must export as "NewObserver".
// ZMS will call this function to create new instances of the plugin.
// Each plugin must export a function with this signature named "NewObserver".
//
// Example:
//
//	func NewObserver() plugin.Observer {
//	    return &MyPlugin{}
//	}
type PluginFactory func() Observer

// PluginInfo contains metadata about a plugin.
// Plugins should export a variable of this type named "PluginInfo" to provide
// information about the plugin to ZMS and users.
//
// Example:
//
//	var PluginInfo = plugin.PluginInfo{
//	    Name:        "my-plugin",
//	    Version:     "1.0.0",
//	    Description: "Example plugin for processing data",
//	    Author:      "Plugin Developer",
//	}
type PluginInfo struct {
	// Name is the human-readable name of the plugin.
	Name string

	// Version is the semantic version of the plugin (e.g., "1.0.0").
	Version string

	// Description provides a brief description of what the plugin does.
	Description string

	// Author identifies who created or maintains the plugin.
	Author string
}
