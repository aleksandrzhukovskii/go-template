package config

type Server struct {
	IP   string `env:"IP" envDefault:"127.0.0.1"`
	Port string `env:"PORT" envDefault:"8000"`
}
