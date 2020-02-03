package types

import (
	"sync"

	"github.com/blho/apexdns/pkg/utils/uuid"

	"github.com/miekg/dns"
)

// Context used for processing every DNS resolve request during server, endpoints and plugins
type Context struct {
	uuid            string
	queryMessage    *dns.Msg
	responseMessage *dns.Msg
	isAbort         bool
	err             error
	payload         map[string]interface{}
	lock            sync.Mutex
}

// NewContext returns a brand new context with query DNS message
func NewContext(queryMessage *dns.Msg) *Context {
	c := &Context{
		uuid:         uuid.Get(),
		queryMessage: queryMessage,
		payload:      make(map[string]interface{}),
	}
	return c
}

func (c *Context) AbortWithErr(err error) {
	c.lock.Lock()
	c.isAbort = true
	c.err = err
	c.lock.Unlock()
}

func (c *Context) Error() error {
	return c.err
}

func (c *Context) IsAbort() bool {
	c.lock.Lock()
	v := c.isAbort
	c.lock.Unlock()
	return v
}

func (c *Context) Set(key string, value interface{}) {
	c.lock.Lock()
	c.payload[key] = value
	c.lock.Unlock()
}

func (c *Context) Get(key string) (interface{}, bool) {
	c.lock.Lock()
	value, ok := c.payload[key]
	c.lock.Unlock()
	return value, ok
}

func (c *Context) GetQueryMessage() *dns.Msg {
	return c.queryMessage
}

func (c *Context) GetUUID() string {
	return c.uuid
}

func (c *Context) SetResponse(msg *dns.Msg) {
	c.responseMessage = msg
}

func (c *Context) GetResponse() *dns.Msg {
	return c.responseMessage
}
