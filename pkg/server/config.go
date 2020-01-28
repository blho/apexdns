package server

import (
	"fmt"

	"github.com/caddyserver/caddy/caddyfile"
)

type RootConfig struct {
	Endpoints []EndpointConfig
	LogLevel  string
}

func NewDefaultRootConfig() *RootConfig {
	return &RootConfig{
		LogLevel: "debug",
	}
}

type EndpointConfig struct {
	Type   string
	Listen string
	// For HTTPS
	KeyFile  string
	CertFile string
}

func ParseRootConfig(block caddyfile.ServerBlock) (*RootConfig, error) {
	c := NewDefaultRootConfig()
	for tokenKey, tokens := range block.Tokens {
		switch tokenKey {
		// HTTP: http :8080
		case "http":
			if len(tokens) == 2 {
				// HTTP endpoint
				// http :8080
				c.Endpoints = append(c.Endpoints, EndpointConfig{
					Type:   tokenKey,
					Listen: tokens[1].Text,
				})
			} else {
				return nil, fmt.Errorf("invalid HTTP endpoint config: %v", tokens)
			}
		case "https":
			if len(tokens) == 4 {
				// HTTPS endpoint
				// https :8080 cert.pem key.pem
				c.Endpoints = append(c.Endpoints, EndpointConfig{
					Type:     tokenKey,
					Listen:   tokens[1].Text,
					CertFile: tokens[2].Text,
					KeyFile:  tokens[3].Text,
				})
			} else {
				return nil, fmt.Errorf("invalid HTTP endpoint config: %v", tokens)
			}
		case "log":
			if len(tokens) == 2 {
				c.LogLevel = tokens[1].Text
			} else {
				return nil, fmt.Errorf("invalid log config: %v", tokens)
			}
		}
	}
	return c, nil
}
