package cache

import (
	"encoding/binary"
	"hash/fnv"
	"time"

	"github.com/miekg/dns"
)

type record struct {
	dns.Msg
	storedAt time.Time
}

func updateRRTTLFromCache(rrs []dns.RR, storedAt time.Time) {
	since := uint32(time.Since(storedAt).Seconds())
	for _, rr := range rrs {
		rr.Header().Ttl -= since
	}
}

func (r record) Get() *dns.Msg {
	shadow := r.Msg.Copy()
	for _, rrs := range [][]dns.RR{
		shadow.Answer,
		shadow.Ns,
		shadow.Extra,
	} {
		updateRRTTLFromCache(rrs, r.storedAt)
	}
	return shadow
}

func getCacheKey(subnet []byte, m *dns.Msg) uint64 {
	h := fnv.New64()
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, m.Question[0].Qtype)
	h.Write(b)
	var c byte
	for i := range m.Question[0].Name {
		c = m.Question[0].Name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		h.Write([]byte{c})
	}
	h.Write(subnet)
	return h.Sum64()
}
