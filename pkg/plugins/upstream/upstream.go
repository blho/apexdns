package upstream

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/proxy"
)

type ups struct {
	net             string
	addr            string
	socks5ProxyAddr string
	socks5DialFunc  proxy.Dialer
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
	if socks5ProxyAddr != "" {
		socks5Proxy, err := proxy.SOCKS5("tcp", u.socks5ProxyAddr, nil, nil)
		if err != nil {
			return nil, err
		}
		u.socks5DialFunc = socks5Proxy
	}
	err := u.setupDNSClient()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (u *ups) getNetwork() string {
	switch u.net {
	case "tcp-tls":
		return "tcp"
	default:
		return u.net
	}
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
	u.dnsClient = dnsClient
	return nil
}

func (u *ups) getConn() (net.Conn, error) {
	dialer := net.Dialer{Timeout: u.timeout}
	dialFunc := dialer.Dial
	network := u.getNetwork()
	if u.socks5DialFunc != nil {
		dialFunc = u.socks5DialFunc.Dial
		network = "tcp"
	}
	return dialFunc(network, u.addr)
}

func (u *ups) exchangeWithCoon(conn net.Conn, msg *dns.Msg) (*dns.Msg, time.Duration, error) {
	startAt := time.Now()
	c := &dns.Conn{
		Conn: conn,
	}
	if u.getNetwork() == "udp" {
		c.UDPSize = dns.MaxMsgSize
	}
	if err := c.WriteMsg(msg); err != nil {
		return nil, time.Since(startAt), err
	}
	response, err := c.ReadMsg()
	return response, time.Since(startAt), err
}

func (u *ups) exchangeViaClient(ctx context.Context, msg *dns.Msg) (r *dns.Msg, rtt time.Duration, err error) {
	r, rtt, err = u.dnsClient.ExchangeContext(ctx, msg, u.addr)
	return r, rtt, err
}

func (u *ups) exchangeViaConn(msg *dns.Msg) (*dns.Msg, time.Duration, error) {
	// TODO(@oif): Add pool support
	conn, err := u.getConn()
	if err != nil {
		return nil, 0, err
	}
	defer conn.Close()
	return u.exchangeWithCoon(conn, msg)
}

func (u *ups) Exchange(msg *dns.Msg) (*dns.Msg, time.Duration, error) {
	response, rtt, err := u.exchangeViaConn(msg)
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
	if err == io.EOF {
		err = nil
	}
	return response, rtt, err
}
