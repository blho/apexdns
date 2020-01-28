package types

import (
	"github.com/caddyserver/caddy/caddyfile"
	"github.com/sirupsen/logrus"
)

type HandleFunc func(*Context)

type PluginSetupFunc func(PluginConfig) (Plugin, error)

type Plugin interface {
	Name() string
	Handle(*Context)
}

type PluginInitializer struct {
	Name        string
	Description string
	SetupFunc   PluginSetupFunc
}

type PluginConfig struct {
	Logger *logrus.Entry
	caddyfile.Dispenser
}
