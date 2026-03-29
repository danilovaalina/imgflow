package config

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Addr        string `mapstructure:"addr"`
	DatabaseURL string `mapstructure:"db_url"`

	// Настройки MinIO
	MinIOEndpoint  string `mapstructure:"minio_endpoint"`
	MinIOAccessKey string `mapstructure:"minio_access_key"`
	MinIOSecretKey string `mapstructure:"minio_secret_key"`
	MinIOBucket    string `mapstructure:"minio_bucket"`

	// Настройки Kafka
	KafkaBrokers []string `mapstructure:"kafka_brokers"`
	KafkaTopic   string   `mapstructure:"kafka_topic"`
	KafkaGroupID string   `mapstructure:"kafka_group_id"`
}

func Load() (Config, error) {
	v := viper.New()

	v.AddConfigPath(".")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Сначала читаем YAML (основные настройки)
	if err := v.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return Config{}, errors.WithStack(err)
		}
	}

	// Мержим .env (если он есть, он перезапишет YAML)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.MergeInConfig() // Ошибку игнорим, файла может не быть

	// Включаем OS ENV (переменные из Docker/Terminal)
	v.AutomaticEnv()
	// Это свяжет "db_url" из тегов с "DB_URL" из системы
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, errors.WithDetail(err, "unable to decode into struct")
	}

	return cfg, nil
}
