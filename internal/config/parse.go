package config

import (
	"os"

	"github.com/caarlos0/env/v11"
)

func New() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.SqLite.Path == "" {
		cfg.SqLite.Path = os.TempDir() + "/db"
	}
	return cfg, nil
}
