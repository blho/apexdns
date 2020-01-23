package server

import (
	"net"
	"net/http"
	"strings"
)

func GetClientIPFromRequest(r *http.Request) net.IP {
	for _, IPFn := range []func() string{
		func() string {
			if XForwardedFor := r.Header.Get("X-Forwarded-For"); XForwardedFor != "" {
				return strings.SplitN(XForwardedFor, ",", 1)[0]
			}
			return ""
		},
		func() string {
			return r.Header.Get("X-Real-IP")
		},
		func() string {
			portIndex := strings.IndexByte(r.RemoteAddr, ':')
			if portIndex > 0 {
				return r.RemoteAddr[:portIndex]
			}
			return r.RemoteAddr
		},
	} {
		IPStr := IPFn()
		if IPStr == "" {
			continue
		}
		return net.ParseIP(strings.TrimSpace(IPStr))
	}
	return nil
}
