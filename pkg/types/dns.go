package types

import (
	"fmt"
	"github.com/miekg/dns"
	"strings"
)

type DNSResponse struct {
	Status             uint32        `json:"Status"`
	Truncated          bool          `json:"TC"`
	RecursionDesired   bool          `json:"RD"`
	RecursionAvailable bool          `json:"RA"`
	AuthenticatedData  bool          `json:"AD"`
	CheckingDisabled   bool          `json:"CD"`
	Question           []DNSQuestion `json:"Question"`
	Answer             []DNSRR       `json:"Answer"`
	Authority          []DNSRR       `json:"Authority,omitempty"`
	Additional         []DNSRR       `json:"Additional,omitempty"`
	Comment            string        `json:"Comment,omitempty"`
	EdnsClientSubnet   string        `json:"edns_client_subnet,omitempty"`
}

type DNSQuestion struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
}
type DNSRR struct {
	DNSQuestion
	TTL  uint32 `json:"TTL"`
	Data string `json:"data"`
}

func ParseDNSResponseFromMessage(msg *dns.Msg) DNSResponse {
	var resp DNSResponse
	resp.Status = uint32(msg.Rcode)
	resp.Truncated = msg.Truncated
	resp.RecursionDesired = msg.RecursionDesired
	resp.RecursionAvailable = msg.RecursionAvailable
	resp.AuthenticatedData = msg.AuthenticatedData
	resp.CheckingDisabled = msg.CheckingDisabled

	resp.Question = make([]DNSQuestion, len(msg.Question))
	for i, ques := range msg.Question {
		resp.Question[i] = DNSQuestion{
			Name: ques.Name,
			Type: ques.Qtype,
		}
	}
	resp.Answer = make([]DNSRR, len(msg.Answer))
	for i, ans := range msg.Answer {
		resp.Answer[i] = parseDNSRR(ans)
	}
	resp.Authority = make([]DNSRR, len(msg.Ns))
	for i, ns := range msg.Ns {
		resp.Authority[i] = parseDNSRR(ns)
	}
	if edns0 := msg.IsEdns0(); edns0 != nil {
		for _, opt := range edns0.Option {
			if o, ok := opt.(*dns.EDNS0_SUBNET); ok {
				resp.EdnsClientSubnet = fmt.Sprintf("%s/%d", o.Address, o.SourceNetmask)
			}
		}
	}
	return resp
}

func parseDNSRR(raw dns.RR) DNSRR {
	// Which defined 5 segments with separator `\t`
	// e.g "apexdns.io IN CNAME homepage.cdn.apexdns.io"
	rr := DNSRR{
		DNSQuestion: DNSQuestion{
			Name: raw.Header().Name,
			Type: raw.Header().Rrtype,
		},
		TTL: raw.Header().Ttl,
	}
	segments := strings.SplitN(raw.String(), "\t", 5)
	if len(segments) == 5 {
		rr.Data = segments[4]
	}
	return rr
}
