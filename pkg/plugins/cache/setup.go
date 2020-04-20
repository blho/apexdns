package cache

import (
	"errors"

	"github.com/blho/apexdns/pkg/server"
	"github.com/blho/apexdns/pkg/types"
)

const (
	Name = "cache"
)

func init() {
	server.RegisterPlugin(types.PluginInitializer{
		Name:        Name,
		Description: "DNS query cache",
		SetupFunc: func(conf types.PluginConfig) (types.Plugin, error) {
			return parse(conf)
		},
	})
}

func parse(conf types.PluginConfig) (*plugin, error) {
	if !conf.Next() {
		return nil, errors.New("invalid plugin config")
	}
	var (
		plug = New()
	)
	plug.logger = conf.Logger.WithField("plugin", Name)
	plug.logger.Info("Initialized cache plugin")
	return plug, nil
}
