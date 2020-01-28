package upstream

import (
	"errors"
	"fmt"
	"github.com/blho/apexdns/pkg/server"
	"github.com/blho/apexdns/pkg/types"
	"time"
)

const (
	Name = "upstream"
)

func init() {
	server.RegisterPlugin(types.PluginInitializer{
		Name:        Name,
		Description: "Multi-upstream DNS resolve plugin",
		SetupFunc: func(conf types.PluginConfig) (plugin types.Plugin, err error) {
			return parse(conf)
		},
	})
}

func parse(conf types.PluginConfig) (*Plugin, error) {
	if !conf.Next() {
		return nil, errors.New("invalid plugin config")
	}
	var (
		plug = New()
	)
	plug.logger = conf.Logger.WithField("plugin", Name)
	args := conf.RemainingArgs()
	if len(args) > 0 {
		timeout, err := time.ParseDuration(args[0])
		if err != nil {
			return nil, err
		}
		plug.timeout = timeout
		plug.logger.WithField("timeout", timeout).Debug("Get DNS client timeout")
	}
	for conf.NextBlock() {
		switch kind := conf.Val(); kind {
		case "tcp":
			args := conf.RemainingArgs()
			if len(args) == 0 {
				return nil, errors.New("upstream is required")
			}
			plug.logger.Debugf("Added TCP upstream: %s", args[0])
			plug.upstreams = append(plug.upstreams, &ups{
				net:  kind,
				addr: args[0],
			})
		case "udp":
			args := conf.RemainingArgs()
			if len(args) == 0 {
				return nil, errors.New("upstream is required")
			}
			plug.logger.Debugf("Added UDP upstream: %s", args[0])
			plug.upstreams = append(plug.upstreams, &ups{
				net:  kind,
				addr: args[0],
			})
		case "tcp-tls":
			args := conf.RemainingArgs()
			if len(args) == 0 {
				return nil, errors.New("upstream is required")
			}
			plug.logger.Debugf("Added TCP-TLS upstream: %s", args[0])
			plug.upstreams = append(plug.upstreams, &ups{
				net:  kind,
				addr: args[0],
			})
		default:
			return nil, fmt.Errorf("unknown config in upstream: %s %v", conf.Val(), conf.RemainingArgs())
		}
	}

	plug.initialize()
	plug.logger.Info("Initialized upstream plugin")
	return plug, nil
}
