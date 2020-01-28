package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/blho/apexdns/pkg/types"

	"github.com/caddyserver/caddy/caddyfile"
	"github.com/miekg/dns"
	"github.com/oif/gokit/logs"
	"github.com/sirupsen/logrus"
)

type Server struct {
	endpoints     []Endpoint
	logger        *logrus.Logger
	opts          Options
	conf          RootConfig
	zoneConfigMap map[string]types.ZoneConfig
	// DNS query clients
	udpClient *dns.Client
	tcpClient *dns.Client
	tlsClient *dns.Client
}

func New(opts Options) (*Server, error) {
	timeout := time.Second * 3
	s := &Server{
		zoneConfigMap: make(map[string]types.ZoneConfig),
		opts:          opts,
		udpClient: &dns.Client{
			Net:     "udp",
			UDPSize: dns.DefaultMsgSize,
			Timeout: timeout,
		},
		tcpClient: &dns.Client{
			Net:     "tcp",
			Timeout: timeout,
		},
		tlsClient: &dns.Client{
			Net:     "tcp-tls",
			Timeout: timeout,
		},
	}
	s.logger = logs.MustSetup()
	if err := s.setup(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) setup() error {
	for _, setupFunc := range []func() error{
		s.loadConfigFile,
		s.setupEndpoints,
	} {
		if err := setupFunc(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) setupEndpoints() error {
	var (
		endpoint Endpoint
		err      error
	)
	for _, endpointConfig := range s.conf.Endpoints {
		switch endpointConfig.Type {
		case "http", "https":
			s.logger.WithFields(logrus.Fields{
				"listen":   endpointConfig.Listen,
				"certFile": endpointConfig.CertFile,
				"keyFile":  endpointConfig.KeyFile,
			}).Info("Setting up HTTP(S) endpoint")
			endpoint, err = NewHTTPEndpoint(endpointConfig.Listen, endpointConfig.CertFile, endpointConfig.KeyFile, s.handleContext)
		default:
			return fmt.Errorf("unsupported endpoint: %s", endpointConfig.Type)
		}
		if err != nil {
			return err
		}
		s.endpoints = append(s.endpoints, endpoint)
	}
	return nil
}

func (s *Server) loadConfigFile() error {
	configFile, err := ioutil.ReadFile(s.opts.ConfigPath)
	if err != nil {
		return err
	}
	blocks, err := caddyfile.Parse(s.opts.ConfigPath, bytes.NewReader(configFile), nil)
	if err != nil {
		return err
	}
	for _, block := range blocks {
		// Get the first key of block
		zone := block.Keys[0]
		if zone == "apexdns" {
			// Is a root config
			rootConfig, err := ParseRootConfig(block)
			if err != nil {
				return err
			}
			s.conf = *rootConfig
		} else if dns.IsFqdn(zone) {
			zoneConfig, err := ParseZoneConfig(block)
			if err != nil {
				return err
			}
			s.zoneConfigMap[zone] = *zoneConfig
		} else {
			return fmt.Errorf("invalid config: %s", zone)
		}
	}
	return nil
}

func (s *Server) handleContext(ctx *types.Context) {
	logger := s.logger.WithField("UUID", ctx.GetUUID())
	logger.WithField("msg", ctx.GetQueryMessage().String()).Info("Handling context")
	if err := ctx.Error(); err != nil {
		logger.WithError(err).Warn("Unable to handle context")
		return
	}
	response, _, err := s.tlsClient.Exchange(ctx.GetQueryMessage(), "dns.google:853")
	if err != nil {
		logger.WithError(err).Error("Unable to exchange query")
		ctx.AbortWithErr(err)
		return
	}
	ctx.SetResponse(response)
}

func (s *Server) Run() {
	for _, endpoint := range s.endpoints {
		go func() {
			err := endpoint.Run()
			if err != nil {
				s.logger.WithError(err).Fatal("Failed to run endpoint")
			}
		}()
	}
}

func (s *Server) Close() error {
	for _, endpoint := range s.endpoints {
		_ = endpoint.Close()
	}
	return nil
}
