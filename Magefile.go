//go:build mage
// +build mage

package main

import (
	"github.com/grafana/grafana-plugin-sdk-go/build"
)

// Default builds the plugin
func Default() error {
	return build.BuildAll()
}
