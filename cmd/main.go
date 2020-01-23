package main

import (
	"os"

	"github.com/blho/apexdns/cmd/server"
	"github.com/blho/apexdns/cmd/version"

	"github.com/spf13/cobra"
)

func main() {
	setupRuntime()

	root := &cobra.Command{
		Use: "apexdns",
	}
	for _, cmd := range []*cobra.Command{
		version.NewCommand(),
		server.NewCommand(),
	} {
		root.AddCommand(cmd)
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
