package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string `env:"APP_ENV" env-default:"production"`
	LogLevel string `env:"LOG_LEVEL" env-default:"info"`

	Telegram TelegramConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
}

type TelegramConfig struct {
	Token string `env:"TELEGRAM_BOT_TOKEN" env-required:"true"`
}

type DatabaseConfig struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}

type RedisConfig struct {
	Addr string `env:"REDIS_ADDR" env-default:"localhost:6379"`
}

type KafkaConfig struct {
	Brokers []string `env:"KAFKA_BROKERS" env-default:"localhost:9092" env-separator:","`
}

func MustLoad() *Config {
	var cfg Config

	if _, err := os.Stat(".env"); err == nil {
		if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
			log.Fatalf("Ошибка чтения .env файла: %s", err)
		}
	} else {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("Ошибка чтения переменных окружения: %s", err)
		}
	}
	return &cfg
}
