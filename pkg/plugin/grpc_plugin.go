// Package plugin provides the gRPC-based plugin interface for ZMS observers.
//
// This file defines the go-plugin wrapper for the gRPC observer service.
// Plugins run as separate processes and communicate via gRPC.
package plugin

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"zms.szuro.net/pkg/proto"
)

// Handshake is the shared configuration between ZMS and plugins.
// This must match exactly between the main application and all plugins
// to ensure compatibility.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ZMS_PLUGIN",
	MagicCookieValue: "zabbix_metric_shipper",
}

// ObserverPlugin is the implementation of the plugin.Plugin interface
// for HashiCorp go-plugin. This handles the gRPC server/client setup.
type ObserverPlugin struct {
	plugin.Plugin
	// Impl is the concrete implementation of the observer
	Impl proto.ObserverServiceServer
}

// GRPCServer registers the observer implementation with the gRPC server.
// This is called by go-plugin when starting the plugin process.
func (p *ObserverPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterObserverServiceServer(s, p.Impl)
	return nil
}

// GRPCClient creates a client that communicates with the plugin.
// This is called by the main ZMS application when connecting to a plugin.
func (p *ObserverPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return proto.NewObserverServiceClient(c), nil
}
