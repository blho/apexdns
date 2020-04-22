package upstream

import (
	"github.com/blho/apexdns/pkg/types"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

type Plugin struct {
	logger    *logrus.Entry
	udpClient *dns.Client
	tcpClient *dns.Client
	tlsClient *dns.Client
	upstreams []*ups
}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) initialize() {}

func (p *Plugin) Name() string {
	return Name
}

func (p *Plugin) Tail(*types.Context) {}

func (p *Plugin) Handle(ctx *types.Context) {
	response, rtt, err := p.bestUpstream().Exchange(ctx.GetQueryMessage())
	if err != nil {
		p.logger.WithError(err).Error("Unable to exchange query")
		ctx.AbortWithErr(err)
		return
	}
	ctx.GetLogger(p.logger).Debugf("Exchanged %s", rtt)
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
