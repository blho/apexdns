package upstream

import (
	"context"
	"time"

	"github.com/blho/apexdns/pkg/types"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

type ups struct {
	net  string
	addr string
	srtt float64
}

func (u *ups) srttAttenuation() {
	u.srtt *= 0.98
}

func (u *ups) exchange(ctx context.Context, client *dns.Client, msg *dns.Msg) (r *dns.Msg, rtt time.Duration, err error) {
	r, rtt, err = client.ExchangeContext(ctx, msg, u.addr)
	go func(ms float64, hasError bool) {
		if hasError {
			u.srtt = u.srtt + 200
			return
		}
		if ms > 300 {
			ms = 300
		}
		u.srtt = u.srtt*0.7 + ms*0.3
	}(float64(rtt/time.Millisecond), err != nil)
	return r, rtt, err
}

type Plugin struct {
	logger    *logrus.Entry
	timeout   time.Duration
	udpClient *dns.Client
	tcpClient *dns.Client
	tlsClient *dns.Client
	upstreams []*ups
}

func New() *Plugin {
	return &Plugin{
		timeout: time.Second * 5,
	}
}

func (p *Plugin) initialize() {
	p.udpClient = &dns.Client{
		Net:     "udp",
		UDPSize: dns.MaxMsgSize,
		Timeout: p.timeout,
	}
	p.tcpClient = &dns.Client{
		Net:     "tcp",
		Timeout: p.timeout,
	}
	p.tlsClient = &dns.Client{
		Net:     "tcp-tls",
		Timeout: p.timeout,
	}
}

func (p *Plugin) Name() string {
	return Name
}

func (p *Plugin) Tail(*types.Context) {}

func (p *Plugin) Handle(ctx *types.Context) {
	u := p.bestUpstream()
	dnsClient := p.tcpClient
	switch u.net {
	case "tcp":
		dnsClient = p.tcpClient
	case "udp":
		dnsClient = p.udpClient
	case "tcp-tls":
		dnsClient = p.tlsClient
	}
	response, _, err := u.exchange(context.Background(), dnsClient, ctx.GetQueryMessage())
	if err != nil {
		p.logger.WithError(err).Error("Unable to exchange query")
		ctx.AbortWithErr(err)
		return
	}
	ctx.SetResponse(response)
}

func (p *Plugin) bestUpstream() *ups {
	best := 0
	for i := 0; i < len(p.upstreams); i++ {
		if p.upstreams[i].srtt < p.upstreams[best].srtt {
			best = i
		}
	}
	go func(selected int) { // lost decay
		for i := 0; i < len(p.upstreams); i++ {
			if i != selected {
				p.upstreams[i].srttAttenuation()
			}
		}
	}(best)

	return p.upstreams[best]
}
