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
//	import "zms.szuro.net/pkg/plugin"
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
