package types

import (
	"github.com/miekg/dns"
	"sync"
)

type Context struct {
	uuid            string
	queryMessage    *dns.Msg
	responseMessage *dns.Msg
	isAbort         bool
	err             error
	payload         map[string]interface{}
	lock            sync.Mutex
}

func NewContext(uuid string, queryMessage *dns.Msg) *Context {
	c := &Context{
		uuid:         uuid,
		queryMessage: queryMessage,
		payload:      make(map[string]interface{}),
	}
	return c
}

func (c *Context) Abort() {
	c.lock.Lock()
	c.isAbort = true
	c.lock.Unlock()
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
