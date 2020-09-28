package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "vent",
	Short: "vent creates a websocket tunnel between among us instances",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level, err := log.ParseLevel(viper.GetString("log.level"))

		if err != nil {
			panic(err)
		}

		log.SetFormatter(&log.TextFormatter{
			ForceColors: viper.GetBool("log.colors"),
		})
		log.SetOutput(os.Stdout)
		log.SetLevel(level)
	},
}

func Execute() {
	// Allow running from explorer
	cobra.MousetrapHelpText = ""

	// Execute client command as default
	cmd, _, err := rootCmd.Find(os.Args[1:])
	if (len(os.Args) <= 1 || os.Args[1] != "help") && (err != nil || cmd == rootCmd) {
		args := append([]string{"client"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.PersistentFlags().String("log", "info", "The log level to output")
	rootCmd.PersistentFlags().Bool("colors", true, "Log output with colors")
	rootCmd.PersistentFlags().Int("port", 56217, "Websocket port")
	rootCmd.PersistentFlags().Int("game-port", 22023, "Game server port")

	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log"))
	_ = viper.BindPFlag("log.colors", rootCmd.PersistentFlags().Lookup("colors"))
	_ = viper.BindPFlag("socket.port", rootCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("server.port", rootCmd.PersistentFlags().Lookup("game-port"))

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("tunnel")
	_ = viper.ReadInConfig()
}
