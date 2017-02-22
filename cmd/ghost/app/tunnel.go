package app

import "github.com/spf13/cobra"

var TunCmd = &cobra.Command{
	Use:   "tun",
	Short: "tunnel",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tunMain()
	},
}

func tunMain() error {
	return nil
}
