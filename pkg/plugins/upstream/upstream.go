package upstream

import (
	"context"
	"time"

	"github.com/miekg/dns"
)

type ups struct {
	net             string
	addr            string
	socks5ProxyAddr string
	timeout         time.Duration
	srtt            float64
	dnsClient       *dns.Client
}

func newUpstream(net, addr, socks5ProxyAddr string, timeout time.Duration) (*ups, error) {
	u := &ups{
		net:             net,
		addr:            addr,
		timeout:         timeout,
		socks5ProxyAddr: socks5ProxyAddr,
		srtt:            0,
		dnsClient:       nil,
	}
	err := u.setupDNSClient()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (u *ups) srttAttenuation() {
	u.srtt *= 0.98
}

func (u *ups) setupDNSClient() error {
	// Initialize DNS client
	dnsClient := &dns.Client{
		Net:     u.net,
		Timeout: u.timeout,
	}
	if u.net == "udp" {
		dnsClient.UDPSize = dns.MaxMsgSize
	}
	// Check if need socks5 proxy
	if u.socks5ProxyAddr != "" {
	}
	u.dnsClient = dnsClient
	return nil
}

func (u *ups) exchange(ctx context.Context, msg *dns.Msg) (r *dns.Msg, rtt time.Duration, err error) {
	r, rtt, err = u.dnsClient.ExchangeContext(ctx, msg, u.addr)
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
