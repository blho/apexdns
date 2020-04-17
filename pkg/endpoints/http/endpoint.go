package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/blho/apexdns/pkg/constant"
	"github.com/blho/apexdns/pkg/types"
	"github.com/miekg/dns"
	"golang.org/x/net/idna"
)

type Endpoint struct {
	httpServer *http.Server
	certFile   string
	keyFile    string
	stopCh     chan struct{}
	userAgent  string
	handler    types.ContextHandler
}

func New(listenAddress string, certFile, keyFile string, handler types.ContextHandler) (*Endpoint, error) {
	e := &Endpoint{
		certFile:  certFile,
		keyFile:   keyFile,
		stopCh:    make(chan struct{}),
		userAgent: "ApexDNS",
		handler:   handler,
	}
	httpServer := &http.Server{
		Addr:    listenAddress,
		Handler: e,
	}
	e.httpServer = httpServer
	return e, nil
}

func (e *Endpoint) Run() error {
	if e.certFile != "" || e.keyFile != "" {
		return e.httpServer.ListenAndServeTLS(e.certFile, e.keyFile)
	}
	return e.httpServer.ListenAndServe()
}

func (e *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.Header().Set("Server", e.userAgent)
	w.Header().Set("X-Powered-By", e.userAgent)

	if r.Method == "OPTIONS" {
		w.Header().Set("Content-Length", "0")
		return
	}

	if r.Form == nil {
		// Max body size 16MB
		r.ParseMultipartForm(16 << 20)
	}

	var ctx *types.Context
	for _, parse := range []func(*http.Request) *types.Context{
		ParseGoogleDoHProtocol,
		ParseIETFDoHProtocol,
	} {
		ctx = parse(r)
		if ctx != nil {
			break
		}
	}
	if ctx == nil {
		// Mismatch DoH protocol
		w.Write([]byte(`{"error":"unknown DoH protocol"}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Handle with context
	e.handler(ctx)
	// Check which content type should response
	// Default as JSON
	contentType := constant.ContentTypeApplicationJSON
	if ct := r.FormValue("ct"); ct != "" {
		// Try Google Protocol
		contentType = ct
	} else {
		// Try Accept header
		for _, acceptContentType := range strings.Split(r.Header.Get("Accept"), ",") {
			acceptContentType = strings.SplitN(acceptContentType, ";", 2)[0]
			if acceptContentType == constant.ContentTypeApplicationDNSJSON ||
				acceptContentType == constant.ContentTypeApplicationJSON ||
				acceptContentType == constant.ContentTypeApplicationXJavascript ||
				acceptContentType == constant.ContentTypeApplicationDNSMessage ||
				acceptContentType == constant.ContentTypeApplicationUDPWireFormat {
				contentType = acceptContentType
				break
			}
		}
	}
	// TODO(@oif): Check `CheckingDisabled` and validate DNSSEC
	// Confirm response content type
	switch contentType {
	case constant.ContentTypeApplicationDNSMessage, constant.ContentTypeApplicationUDPWireFormat:
		e.responseOnDNSMsg(ctx, w)
	default:
		// Default use JSON
		fallthrough
	case constant.ContentTypeApplicationXJavascript, constant.ContentTypeApplicationJSON, constant.ContentTypeApplicationDNSJSON:
		e.responseOnJSON(ctx, w)
	}
}

// TODO(@oif): Graceful shutdown
func (e *Endpoint) Close() error {
	close(e.stopCh)
	if e.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	if err := e.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func checkDisabledECS(r *http.Request) bool {
	if r.FormValue("de") != "" {
		return true
	}
	return false
}

// Reference https://developers.google.com/speed/public-dns/docs/doh/json
func ParseGoogleDoHProtocol(r *http.Request) *types.Context {
	domainName := r.FormValue("name")
	if domainName == "" {
		return nil
	}
	msg := new(dns.Msg)
	ctx := types.NewContext(GetClientIPFromRequest(r), msg)
	if punycode, err := idna.ToASCII(domainName); err == nil {
		domainName = punycode
	} else {
		ctx.AbortWithErr(err)
		return ctx
	}

	// Default to A record
	rrType := dns.TypeA
	rrTypeStr := r.FormValue("type")
	if rrTypeStr != "" {
		// try uint16
		if rt, err := strconv.ParseUint(rrTypeStr, 10, 16); err == nil {
			rrType = uint16(rt)
		} else if rt, ok := dns.StringToType[strings.ToUpper(rrTypeStr)]; ok {
			// try canonical string
			rrType = rt
		} else {
			// invalid RR type
			ctx.AbortWithErr(fmt.Errorf("invalid RR type: %s", rrTypeStr))
			return ctx
		}
	}

	cdStr := r.FormValue("cd")
	checkingDisabled, ok := parseGenericBool(cdStr, false)
	if !ok {
		ctx.AbortWithErr(fmt.Errorf("invalid DNSSEC checking disabled(cd): %s", cdStr))
		return ctx
	}

	doStr := r.FormValue("do")
	includeDNSSECRecord, ok := parseGenericBool(doStr, false)
	if !ok {
		ctx.AbortWithErr(fmt.Errorf("invalid do: %s", doStr))
		return ctx
	}
	// Set msg
	msg.SetQuestion(dns.Fqdn(domainName), rrType)
	msg.CheckingDisabled = checkingDisabled
	opt := new(dns.OPT)
	opt.Hdr.Name = "."
	opt.Hdr.Rrtype = dns.TypeOPT
	opt.SetUDPSize(dns.DefaultMsgSize)
	opt.SetDo(includeDNSSECRecord)
	if !checkDisabledECS(r) {
		// ECS
		var (
			// Unspecified netmask size set to 256
			ednsNetmask   = uint16(256)
			ednsIPAddress net.IP
		)

		if ednsClientSubnet := r.FormValue("edns_client_subnet"); ednsClientSubnet == "" {
			// Get IP from request
			if ip := ctx.ClientIP(); ip != nil {
				ednsIPAddress = ip
			}
		} else {
			subnetSlashIndex := strings.IndexByte(ednsClientSubnet, '/')
			if subnetSlashIndex > 0 {
				ip := net.ParseIP(ednsClientSubnet[:subnetSlashIndex])
				if ip == nil {
					// Invalid ECS IP
					ctx.AbortWithErr(fmt.Errorf("invalid ECS IP(edns_client_subnet): %s", ednsClientSubnet))
					return ctx
				}
				ednsIPAddress = ip
				netmask, err := strconv.ParseUint(ednsClientSubnet[subnetSlashIndex+1:], 10, 8)
				if err != nil {
					// Invalid subnet
					ctx.AbortWithErr(fmt.Errorf("invalid ECS subnet(edns_client_subnet): %s", ednsClientSubnet))
					return ctx
				}
				ednsNetmask = uint16(netmask)
			} else {
				ip := net.ParseIP(ednsClientSubnet)
				if ip == nil {
					// Invalid ECS IP
					ctx.AbortWithErr(fmt.Errorf("invalid ECS IP(edns_client_subnet): %s", ednsClientSubnet))
					return ctx
				}
				ednsIPAddress = ip
			}
		}

		if ednsIPAddress != nil {
			family := uint16(1)
			if ednsIPAddress.To4() == nil {
				family = 2
			}
			var netmask uint8
			if ednsNetmask == 256 {
				// Default IPv4 subnet /24, IPv6 subnet /56
				if family == 1 {
					netmask = 24
				} else {
					netmask = 56
				}
			} else {
				netmask = uint8(ednsNetmask)
			}
			opt.Option = append(opt.Option, &dns.EDNS0_SUBNET{
				Code:          dns.EDNS0SUBNET,
				Family:        family,
				SourceNetmask: netmask,
				SourceScope:   0,
				Address:       ednsIPAddress,
			})
		}
	}
	msg.Extra = append(msg.Extra, opt)
	return ctx
}

// Reference https://www.rfc-editor.org/rfc/rfc8484.html
func ParseIETFDoHProtocol(r *http.Request) *types.Context {
	msg := new(dns.Msg)
	ctx := types.NewContext(GetClientIPFromRequest(r), msg)
	var (
		rawMessage []byte
		err        error
	)
	rawMessageStr := r.FormValue("dns")
	if len(rawMessageStr) > 0 {
		rawMessage, err = base64.RawURLEncoding.DecodeString(rawMessageStr)
	} else {
		rawMessage, err = ioutil.ReadAll(r.Body)
	}
	if err != nil {
		ctx.AbortWithErr(fmt.Errorf("invalid request body: %s", err))
		return ctx
	}
	if len(rawMessage) == 0 {
		return nil
	}
	err = msg.Unpack(rawMessage)
	if err != nil {
		ctx.AbortWithErr(fmt.Errorf("invalid request body: %s", err))
		return ctx
	}
	opt := msg.IsEdns0()
	if opt == nil {
		opt = new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(dns.DefaultMsgSize)
		opt.SetDo(false)
	}
	// Find EDNS0SUBNET
	hasEDNS0SubnetOption := false
	for _, o := range opt.Option {
		if o.Option() == dns.EDNS0SUBNET {
			hasEDNS0SubnetOption = true
			break
		}
	}
	if !checkDisabledECS(r) && !hasEDNS0SubnetOption {
		// Get IP from request
		if ip := ctx.ClientIP(); ip != nil {
			// Default IPv4 subnet /24, IPv6 subnet /56
			family := uint16(1)
			netmask := uint8(24)
			if ip.To4() == nil {
				family = 2
				netmask = 56
			}
			opt.Option = append(opt.Option, &dns.EDNS0_SUBNET{
				Code:          dns.EDNS0SUBNET,
				Family:        family,
				SourceNetmask: netmask,
				SourceScope:   0,
				Address:       ip,
			})
		}
	}
	msg.Extra = append(msg.Extra, opt)
	return ctx
}

func (e *Endpoint) responseOnJSON(ctx *types.Context, w http.ResponseWriter) {
	if err := ctx.Error(); err != nil {
		w.Header().Set("Content-Type", constant.ContentTypeApplicationJSON)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp := ctx.GetResponse()
	if resp == nil {
		w.Header().Set("Content-Type", constant.ContentTypeApplicationJSON)
		w.Write([]byte(`{"error":"internal error"}`))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	payload, err := json.Marshal(ParseDNSResponseFromMessage(resp))
	if err != nil {
		w.Header().Set("Content-Type", constant.ContentTypeApplicationJSON)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", constant.ContentTypeApplicationJSON)
	w.Write(payload)
}

func (e *Endpoint) responseOnDNSMsg(ctx *types.Context, w http.ResponseWriter) {
	if err := ctx.Error(); err != nil {
		w.Header().Set("Content-Type", constant.ContentTypeApplicationJSON)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp := ctx.GetResponse()
	if resp == nil {
		w.Header().Set("Content-Type", constant.ContentTypeApplicationJSON)
		w.Write([]byte(`{"error":"internal error"}`))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	result, err := resp.Pack()
	if err != nil {
		w.Header().Set("Content-Type", constant.ContentTypeApplicationJSON)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", constant.ContentTypeApplicationDNSMessage)
	_, _ = w.Write(result)
}

func parseGenericBool(raw string, defaultValue bool) (result, ok bool) {
	switch raw {
	case "":
		// use default value
		return defaultValue, true
	case "1", "true":
		result = true
	case "0", "false":
		result = false
	default:
		return false, false
	}
	return result, true
}
