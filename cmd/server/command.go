package server

import (
	"syscall"

	_ "github.com/blho/apexdns/pkg/plugins"
	"github.com/blho/apexdns/pkg/server"

	"github.com/oif/gokit/wait"
	"github.com/spf13/cobra"
)

// NewCommand of ApexDNS server
func NewCommand() *cobra.Command {
	opt := server.NewDefaultOptions()
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run ApexDNS server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := Run(opt); err != nil {
				return err
			}
			return nil
		},
	}
	opt.AddFlags(cmd.Flags())
	return cmd
}

// Run will start running and returns error if something unexpected
func Run(opt *server.Options) error {
	s, err := server.New(*opt)
	if err != nil {
		return err
	}
	s.Run()
	// Hold process wait util receive signal below then shutdown
	wait.Signal(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	return s.Close()
}
