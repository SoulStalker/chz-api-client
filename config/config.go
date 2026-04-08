package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Server ServerConfig `yaml:"server"`
	Diadoc DiadocConfig `yaml:"diadoc"`
	CRPT   CRPTConfig   `yaml:"crpt"`
	Signer SignerConfig `yaml:"signer"`
	Log    LogConfig    `yaml:"log"`
}

type ServerConfig struct {
	Addr string `yaml:"addr" env:"SERVER_ADDR" env-default:":8080"`
}

type DiadocConfig struct {
	BaseURL  string `yaml:"base_url"  env:"DIADOC_BASE_URL"  env-default:"https://diadoc-api.kontur.ru"`
	ClientID string `yaml:"client_id" env:"DIADOC_CLIENT_ID"`
}

type CRPTConfig struct {
	BaseURL string `yaml:"base_url" env:"CRPT_BASE_URL" env-default:"https://markirovka.crpt.ru"`
}

type SignerConfig struct {
	Addr string `yaml:"addr" env:"SIGNER_ADDR" env-default:"localhost:50051"`
}

type LogConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
