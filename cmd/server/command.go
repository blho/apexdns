package server

import (
	"syscall"

	"github.com/blho/apexdns/pkg/server"

	"github.com/oif/gokit/wait"
	"github.com/spf13/cobra"
)

// NewCommand of ApexDNS server
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run ApexDNS server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := Run(); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

// Run will start running and returns error if something unexpected
func Run() error {
	s, err := server.New()
	if err != nil {
		return err
	}
	s.Run()
	// Hold process wait util receive signal below then shutdown
	wait.Signal(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	return s.Close()
}
