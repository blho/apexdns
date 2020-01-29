package server

import (
	"fmt"
	"github.com/caddyserver/caddy/caddyfile"

	"github.com/blho/apexdns/pkg/types"

	"github.com/sirupsen/logrus"
)

type Engine struct {
	logger      *logrus.Entry
	pluginChain []types.Plugin
}

func NewEngine(logger *logrus.Entry, tokens map[string][]caddyfile.Token) (*Engine, error) {
	e := new(Engine)
	e.logger = logger
	logger.Info("Setting up zone engine")
	err := e.loadPlugins(tokens)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) loadPlugins(tokens map[string][]caddyfile.Token) error {
	for pluginName, tokens := range tokens {
		pluginInitializer, ok := GetPlugin(pluginName)
		if !ok {
			return fmt.Errorf("plugin `%s` not registered yet", pluginName)
		}
		plugin, err := pluginInitializer.SetupFunc(types.PluginConfig{
			Logger:    e.logger,
			Dispenser: caddyfile.NewDispenserTokens("engine_plugin", tokens),
		})
		if err != nil {
			return err
		}
		e.pluginChain = append(e.pluginChain, plugin)
		e.logger.Infof("Load plugin %s: %s", pluginInitializer.Name, pluginInitializer.Description)
	}
	return nil
}

func (e *Engine) Handle(ctx *types.Context) {
	for _, plugin := range e.pluginChain {
		if ctx.Error() != nil {
			return
		}
		plugin.Handle(ctx)
	}
}
