package config

import "fmt"

type Sqlite struct {
	Path string `env:"STORAGE_PATH"`
}

type MySQL struct {
	Host     string `env:"DB_HOST"`
	Port     int    `env:"DB_PORT" envDefault:"3306"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	Database string `env:"DB_NAME"`
}

func (c MySQL) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.User, c.Password, c.Host, c.Port, c.Database)
}

type Postgres struct {
	Host     string `env:"DB_HOST"`
	Port     int    `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	Database string `env:"DB_NAME"`
}

func (c Postgres) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.User, c.Password, c.Host, c.Port, c.Database)
}

func (c Postgres) GormDNS() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		c.Host, c.User, c.Password, c.Database, c.Port)
}

type Mongo struct {
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	Host     string `env:"DB_HOST"`
	Port     int    `env:"DB_PORT" envDefault:"27017"`
	Database string `env:"DB_NAME"`
}

func (c Mongo) DSN() string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%d", c.User, c.Password, c.Host, c.Port)
}

type Clickhouse struct {
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	Host     string `env:"DB_HOST"`
	Port     int    `env:"DB_PORT" envDefault:"9000"`
	Database string `env:"DB_NAME"`
}
