package server

import (
	"fmt"

	"github.com/blho/apexdns/pkg/types"
)

var (
	registeredEndpointInitializers = make(map[string]types.EndpointInitializer)
)

func RegisterEndpoint(edp types.EndpointInitializer) error {
	_, duplicated := registeredEndpointInitializers[edp.Name]
	if duplicated {
		return fmt.Errorf("duplicated endpoint name: %s", edp.Name)
	}
	registeredEndpointInitializers[edp.Name] = edp
	return nil
}

func GetEndpoint(name string) (types.EndpointInitializer, bool) {
	edp, ok := registeredEndpointInitializers[name]
	return edp, ok
}
