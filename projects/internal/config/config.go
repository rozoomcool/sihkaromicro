package config

import (
	"time"

	"github.com/spf13/viper"
)

type DBConf struct {
	DBDSN           string        `mapstructure:"db-dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type AuthCfg struct {
	ProviderURL string `mapstructure:"providerURL"`
	ClientID    string `mapstructure:"clientID"`
}

type Config struct {
	Env     string `mapstructure:"env"`
	LogFile string `mapstructure:"logFile"`
	GRPC    struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"grpc"`

	DB   DBConf  `mapstructure:"db"`
	Auth AuthCfg `mapstructure:"auth"`
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
