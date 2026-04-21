package config

import (
	"strings"

	"github.com/spf13/viper"
)

type KeycloakCfg struct {
	Url    string `mapstructure:"url"`
	Realm  string `mapstructure:"realm"`
	Client struct {
		ID string `mapstructure:"id"`
	} `mapstructure:"client"`
}

type Config struct {
	Env     string `mapstructure:"env"`
	LogFile string `mapstructure:"logFile"`
	GRPC    struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"grpc"`
	Keycloak KeycloakCfg `mapstructure:"keycloak"`
}

func MustLoad() *Config {
	viper.SetConfigFile("configs/config.yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
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
