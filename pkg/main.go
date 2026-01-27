package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sabio/grafana-sm3-chat-plugin/pkg/plugin"
)

func main() {
	// Create plugin
	p := plugin.NewPlugin()

	// Serve plugin
	if err := datasource.Manage("sabio-sm3-chat", p, datasource.ManageOpts{}); err != nil {
		log.DefaultLogger.Error("Plugin exited with error", "error", err)
		os.Exit(1)
	}
}
