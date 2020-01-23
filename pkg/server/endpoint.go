package server

import (
	"github.com/blho/apexdns/pkg/types"
)

type Endpoint interface {
	Run() error
	Close() error
}

type ContextHandler func(*types.Context)
