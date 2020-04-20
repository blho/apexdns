package cache

import (
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"

	"github.com/blho/apexdns/pkg/types"
)

const (
	contextPayloadMark = "cached_message"
)

type plugin struct {
	logger      *logrus.Entry
	bucketCache *bucketCache
}

func New() *plugin {
	return &plugin{
		bucketCache: newCache(1048576),
	}
}

func (p *plugin) Name() string {
	return Name
}

func (p *plugin) Handle(ctx *types.Context) {
	if ctx.Error() != nil && ctx.GetResponse() != nil {
		return
	}
	// try cache
	logger := ctx.GetLogger(p.logger)
	if response := p.getCache(ctx.ClientIP(), ctx.GetQueryMessage()); response != nil {
		logger.Debug("Hit cache")
		ctx.SetResponse(response)
		ctx.Abort()
	} else {
		logger.Debug("Cache missing")
	}
}

func (p *plugin) Tail(ctx *types.Context) {
	if ctx.Error() != nil || ctx.GetResponse() == nil {
		return
	}
	if _, ok := ctx.Get(contextPayloadMark); ok {
		// Cached response
		return
	}
	// write cache
	msg := ctx.GetResponse()
	clientIP := ctx.ClientIP()
	if msg.Truncated || len(msg.Answer) == 0 {
		return
	}
	p.writeCache(clientIP, msg)
}

func (p *plugin) writeCache(clientIP net.IP, m *dns.Msg) {
	cacheKey := getCacheKey(clientIP[:len(clientIP)-1], m)
	_, ok := p.bucketCache.Get(cacheKey)
	if ok {
		return
	}
	p.bucketCache.Set(cacheKey, record{
		Msg:      *m,
		storedAt: time.Now(),
	}, m.Answer[0].Header().Ttl)
}

func (p *plugin) getCache(clientIP net.IP, m *dns.Msg) *dns.Msg {
	cacheKey := getCacheKey(clientIP[:len(clientIP)-1], m)
	r, ok := p.bucketCache.Get(cacheKey)
	if !ok {
		return nil
	}
	return r.(record).Get()
}
