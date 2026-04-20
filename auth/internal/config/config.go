package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Env     string `mapstructure:"env"`
	LogFile string `mapstructure:"logFile"`
	GRPC    struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"grpc"`
	ProviderURL string `mapstructure:"providerURL"`
	ClientID    string `mapstructure:"clientID"`
}

func MustLoad() *Config {
	viper.SetConfigFile("configs/config.yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic("Error occured when loading configs: " + err.Error())
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic("Error occured when loading configs: " + err.Error())
	}

	return &cfg
}
