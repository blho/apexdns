package http

import (
	"errors"
	"fmt"

	"github.com/blho/apexdns/pkg/server"
	"github.com/blho/apexdns/pkg/types"
)

const (
	Name = "http"
)

func init() {
	server.RegisterEndpoint(types.EndpointInitializer{
		Name:        Name,
		Description: "HTTP(S) resolver endpoint",
		SetupFunc: func(config types.EndpointConfig) (types.Endpoint, error) {
			return parse(config)
		},
	})
}

func parse(c types.EndpointConfig) (endpoint types.Endpoint, err error) {
	if !c.Next() {
		return nil, errors.New("invalid HTTP endpoint config")
	}
	args := c.RemainingArgs()
	var (
		listenAddr string
		certFile   string
		keyFile    string
	)
	switch len(args) {
	case 1:
		// Listen port only
		listenAddr = args[0]
	case 3:
		listenAddr = args[0]
		certFile = args[1]
		keyFile = args[2]
	default:
		return nil, fmt.Errorf("invalid HTTP endpoint arguments: %v", args)
	}
	return New(listenAddr, certFile, keyFile, c.Handler)
}
