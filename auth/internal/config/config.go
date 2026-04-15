package config

import (
	"time"

	"github.com/spf13/viper"
)

type DBConf struct {
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type JWTConf struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

type Config struct {
	Env     string `mapstructure:"env"`
	LogFile string `mapstructure:"logFile"`
	GRPC    struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"grpc"`

	DB DBConf `mapstructure:"db"`

	JWT JWTConf `mapstructure:"jwt"`
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

// func (c *Config) GetConnMaxLifetime() time.Duration {
// 	d, _ := time.ParseDuration(c.DB.ConnMaxLifetime)
// 	return d
// }

// func (c *Config) GetAccessTokenTTL() time.Duration {
// 	d, _ := time.ParseDuration(c.JWT.AccessTokenTTL)
// 	return d
// }

// func (c *Config) GetRefreshTokenTTL() time.Duration {
// 	d, _ := time.ParseDuration(c.JWT.RefreshTokenTTL)
// 	return d
// }
