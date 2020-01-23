package version

import (
	"fmt"

	"github.com/blho/apexdns/pkg/version"

	"github.com/spf13/cobra"
)

// NewCommand to show Matrix version
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(c *cobra.Command, args []string) {
			fmt.Println(version.Get())
		},
	}
	return cmd
}
