package server

import (
	"github.com/spf13/pflag"
)

type Options struct {
	ConfigPath string
}

const (
	DefaultConfigPath = "/etc/apexdns/Apexfile"
)

func NewDefaultOptions() *Options {
	return &Options{
		ConfigPath: DefaultConfigPath,
	}
}

func (o *Options) AddFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&o.ConfigPath, "config-path", "c", DefaultConfigPath, "Config file(Apexfile) path")
}
