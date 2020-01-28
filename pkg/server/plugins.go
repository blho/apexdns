package server

import (
	"fmt"

	"github.com/blho/apexdns/pkg/types"
)

var (
	registeredPluginInitializers = make(map[string]types.PluginInitializer)
)

func RegisterPlugin(plug types.PluginInitializer) error {
	_, duplicated := registeredPluginInitializers[plug.Name]
	if duplicated {
		return fmt.Errorf("duplicated plugin name: %s", plug.Name)
	}
	registeredPluginInitializers[plug.Name] = plug
	return nil
}

func GetPlugin(name string) (types.PluginInitializer, bool) {
	plug, ok := registeredPluginInitializers[name]
	return plug, ok
}
