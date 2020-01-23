package server

import (
	"github.com/blho/apexdns/pkg/types"
	"github.com/miekg/dns"
	"github.com/oif/gokit/logs"
	"github.com/sirupsen/logrus"
	"time"
)

type Server struct {
	endpoints []Endpoint
	logger    *logrus.Logger
	// DNS query clients
	udpClient *dns.Client
	tcpClient *dns.Client
	tlsClient *dns.Client
}

func New() (*Server, error) {
	timeout := time.Second * 3
	s := &Server{
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
	endpoint, err := NewHTTPEndpoint(":8080", "", "", s.handleContext)
	if err != nil {
		return err
	}
	s.endpoints = append(s.endpoints, endpoint)
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
