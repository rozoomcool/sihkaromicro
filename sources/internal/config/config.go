package config

import (
	"os"
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

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
}

type Config struct {
	Env     string `mapstructure:"env"`
	LogFile string `mapstructure:"log_file"`
	GRPC    struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"grpc"`

	DB       DBConf      `mapstructure:"db"`
	Keycloak KeycloakCfg `mapstructure:"keycloak"`
	MinIO    MinIOConfig `mapstructure:"minio"`
	Kafka    KafkaConfig `mapstructure:"kafka"`

	ProjectsUrl string `mapstructure:"PROJECTS_URL"`
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

	// MinIO и Kafka читаем напрямую из env
	cfg.MinIO = MinIOConfig{
		Endpoint:        os.Getenv("MINIO_ENDPOINT"),
		AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
		BucketName:      os.Getenv("MINIO_BUCKET"),
		UseSSL:          false,
	}

	cfg.Kafka = KafkaConfig{
		Brokers: strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		Topic:   os.Getenv("KAFKA_TOPIC"),
	}

	cfg.ProjectsUrl = os.Getenv("PROJECTS_URL")

	return &cfg
}
