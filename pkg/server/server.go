package server

import (
	"bytes"
	"io/ioutil"
	"strings"

	"github.com/blho/apexdns/pkg/types"

	"github.com/caddyserver/caddy/caddyfile"
	"github.com/miekg/dns"
	"github.com/oif/gokit/logs"
	"github.com/sirupsen/logrus"
)

type Server struct {
	endpoints  []types.Endpoint
	logger     *logrus.Logger
	opts       Options
	conf       []caddyfile.ServerBlock
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
	s.conf = blocks
	return nil
}

func (s *Server) setupLogger() error {
	logLevel := "debug"
	for _, block := range s.conf {
		if len(block.Keys) == 0 {
			continue
		}
		if block.Keys[0] == "apexdns" {
			if logConfig, ok := block.Tokens["log"]; ok && len(logConfig) > 1 {
				logLevel = logConfig[1].Text
			}
			break
		}
	}
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	s.logger = logs.MustSetup(logs.WithLogLevel(level), logs.WithSourceHook())
	s.logger.Info("Initialized logger")
	return nil
}

func (s *Server) setupEndpoints() error {
	for _, block := range s.conf {
		if len(block.Keys) == 0 {
			continue
		}
		if block.Keys[0] == "apexdns" {
			for k, v := range block.Tokens {
				endpointInitializer, ok := GetEndpoint(k)
				if !ok {
					s.logger.Debugf("%s is not a endpoint, skip", k)
					continue
				}
				edp, err := endpointInitializer.SetupFunc(types.EndpointConfig{
					Logger:    s.logger.WithField("endpoint", endpointInitializer.Name),
					Handler:   s.handleContext,
					Dispenser: caddyfile.NewDispenserTokens(s.opts.ConfigPath, v),
				})
				if err != nil {
					s.logger.WithError(err).Errorf("Failed to setup endpoint: %s", k)
					return err
				}
				s.endpoints = append(s.endpoints, edp)
			}
			break
		}
	}
	return nil
}

func (s *Server) setupEngine() error {
	for _, block := range s.conf {
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
