package config

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func InitializeConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("tunnel")

	initializeDefaults()

	err := viper.ReadInConfig()
	if err != nil {
		log.Warning(errors.Wrap(err, "config initialized using defaults and environment only!"))
	}

	log.Info("Config initialized")
}

func initializeDefaults() {
	viper.SetDefault("log.level", "info")

	viper.SetDefault("socket.host", "google.com")
	viper.SetDefault("socket.port", 56217)

	viper.SetDefault("server.host", "127.0.0.1")
	viper.SetDefault("server.port", 22023)
	viper.SetDefault("server.name", "Green is impostor")

	viper.SetDefault("broadcast.host", "127.0.0.1")
	viper.SetDefault("broadcast.port", 47777)
}
