package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/plugin"
)

func main() {
	// Create plugin
	p := plugin.NewPlugin()

	// Serve plugin using backend.Manage with ServeOpts
	if err := backend.Manage("sabio-sm3-chat-plugin", backend.ServeOpts{
		CallResourceHandler: p,
	}); err != nil {
		log.DefaultLogger.Error("Plugin exited with error", "error", err)
		os.Exit(1)
	}
}
