package cmd

import (
	"github.com/Vilsol/vent/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	clientCmd.PersistentFlags().String("name", "Green is impostor", "Broadcasted server name")
	clientCmd.PersistentFlags().Int("broadcast-port", 47777, "Broadcasting port")
	clientCmd.PersistentFlags().String("host", "google.com", "Remote tunnel address")
	clientCmd.PersistentFlags().StringSlice("direct", []string{}, "Broadcast directly to provided IP/Subnet")

	_ = viper.BindPFlag("server.name", clientCmd.PersistentFlags().Lookup("name"))
	_ = viper.BindPFlag("broadcast.port", clientCmd.PersistentFlags().Lookup("broadcast-port"))
	_ = viper.BindPFlag("socket.host", clientCmd.PersistentFlags().Lookup("host"))
	_ = viper.BindPFlag("broadcast.direct", clientCmd.PersistentFlags().Lookup("direct"))

	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run client side of the tunnel",
	Run: func(cmd *cobra.Command, args []string) {
		client.RunClient()
	},
}
