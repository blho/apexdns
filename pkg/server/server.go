package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/blho/apexdns/pkg/types"

	"github.com/caddyserver/caddy/caddyfile"
	"github.com/miekg/dns"
	"github.com/oif/gokit/logs"
	"github.com/sirupsen/logrus"
)

type Server struct {
	endpoints  []Endpoint
	logger     *logrus.Logger
	opts       Options
	conf       RootConfig
	zoneEngine map[string]*Engine
}

func New(opts Options) (*Server, error) {
	s := &Server{
		zoneEngine: make(map[string]*Engine),
		opts:       opts,
	}
	if err := s.setup(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) setup() error {
	for _, setupFunc := range []func() error{
		s.loadConfigFile,
		s.setupLogger,
		s.setupEndpoints,
		s.setupEngine,
	} {
		if err := setupFunc(); err != nil {
			return err
		}
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
			break
		}
	}
	return nil
}

func (s *Server) setupLogger() error {
	level, err := logrus.ParseLevel(s.conf.LogLevel)
	if err != nil {
		return err
	}
	s.logger = logs.MustSetup(logs.WithLogLevel(level), logs.WithSourceHook())
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

func (s *Server) setupEngine() error {
	configFile, err := ioutil.ReadFile(s.opts.ConfigPath)
	if err != nil {
		return err
	}
	blocks, err := caddyfile.Parse(s.opts.ConfigPath, bytes.NewReader(configFile), nil)
	if err != nil {
		return err
	}
	for _, block := range blocks {
		if len(block.Keys) == 0 {
			continue
		}
		zone := block.Keys[0]
		if dns.IsFqdn(zone) {
			eng, err := NewEngine(s.logger.WithField("zone", zone), block.Tokens)
			if err != nil {
				return err
			}
			s.zoneEngine[zone] = eng
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
	// Find the engine to handle this context
	eng := s.findBestMatchZoneEngine(ctx.GetQueryMessage().Question[0].Name)
	eng.Handle(ctx)
}

func (s *Server) findBestMatchZoneEngine(fqdn string) *Engine {
	maxMatchLength := 0
	var matchEngine *Engine
	for zone, eng := range s.zoneEngine {
		if strings.HasSuffix(fqdn, zone) && len(zone) > maxMatchLength {
			s.logger.Debugf("Hit zone: %s", zone)
			matchEngine = eng
			maxMatchLength = len(zone)
		}
	}
	return matchEngine
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
