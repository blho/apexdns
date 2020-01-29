package types

import (
	"github.com/caddyserver/caddy/caddyfile"
	"github.com/sirupsen/logrus"
)

type Endpoint interface {
	Run() error
	Close() error
}

type ContextHandler func(*Context)

type EndpointInitializer struct {
	Name        string
	Description string
	SetupFunc   EndpointSetupFunc
}

type EndpointSetupFunc func(EndpointConfig) (Endpoint, error)

type EndpointConfig struct {
	Logger *logrus.Entry
	caddyfile.Dispenser
	Handler ContextHandler
}
