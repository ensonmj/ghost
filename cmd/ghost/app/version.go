package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "unknown"
	buildTime = "unknown"
)

var VerCmd = &cobra.Command{
	Use:   "version",
	Short: "Command version and build time, etc.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stdout, "Version: %s\nBuild Time: %s\n", version, buildTime)
	},
}
