package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type DBConf struct {
	DSN             string        `mapstructure:"dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

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

	DB       DBConf      `mapstructure:"db"`
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
