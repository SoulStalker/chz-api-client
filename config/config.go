package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	CRPT   CRPTConfig   `yaml:"crpt"`
	Signer SignerConfig `yaml:"signer"`
	Log    LogConfig    `yaml:"log"`
}

type ServerConfig struct {
	Addr string `yaml:"addr" env:"SERVER_ADDR" env-default:":8080"`
}

type CRPTConfig struct {
	BaseURL    string `yaml:"base_url"    env:"CRPT_BASE_URL"    env-default:"https://markirovka.crpt.ru"`
	Thumbprint string `yaml:"thumbprint"  env:"CRPT_THUMBPRINT"`
	INN        string `yaml:"inn"         env:"CRPT_INN"`
	MCHD       string `yaml:"mchd"        env:"CRPT_MCHD"`
}

type SignerConfig struct {
	Addr string `yaml:"addr" env:"SIGNER_ADDR" env-default:"localhost:50051"`
}

type LogConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

func MustLoad(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exits: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
