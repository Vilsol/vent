package cmd

import (
	"github.com/Vilsol/vent/host"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	hostCmd.PersistentFlags().String("server", "127.0.0.1", "Address of game server")

	_ = viper.BindPFlag("server.host", hostCmd.PersistentFlags().Lookup("server"))

	rootCmd.AddCommand(hostCmd)
}

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Run host side of the tunnel",
	Run: func(cmd *cobra.Command, args []string) {
		host.RunHost()
	},
}
