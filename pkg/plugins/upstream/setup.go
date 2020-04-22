package upstream

import (
	"errors"
	"fmt"
	"time"

	"github.com/blho/apexdns/pkg/server"
	"github.com/blho/apexdns/pkg/types"
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
	var upstreamTimeout time.Duration
	if len(args) > 0 {
		timeout, err := time.ParseDuration(args[0])
		if err != nil {
			return nil, err
		}
		upstreamTimeout = timeout
		plug.logger.WithField("timeout", timeout).Debug("Get DNS client timeout")
	}
	for conf.NextBlock() {
		switch kind := conf.Val(); kind {
		case "tcp", "udp", "tcp-tls":
			args := conf.RemainingArgs()
			plug.logger.Infof("Adding %s upstream: %v", kind, args)
			upstream, err := parseUpstreamArgs(kind, args, upstreamTimeout)
			if err != nil {
				return nil, err
			}
			plug.upstreams = append(plug.upstreams, upstream)
		default:
			return nil, fmt.Errorf("unknown config in upstream: %s %v", conf.Val(), conf.RemainingArgs())
		}
	}

	plug.initialize()
	plug.logger.Info("Initialized upstream plugin")
	return plug, nil
}

func parseUpstreamArgs(kind string, args []string, timeout time.Duration) (*ups, error) {
	// args upstreamAddr [socks5Addr]
	if len(args) == 0 {
		return nil, errors.New("upstream is required")
	}
	// TODO(@oif): Check address format <IP>:<Port>
	var (
		upstreamAddr    = args[0]
		socks5ProxyAddr string
	)
	if len(args) == 2 {
		socks5ProxyAddr = args[1]
	}
	return newUpstream(kind, upstreamAddr, socks5ProxyAddr, timeout)
}
