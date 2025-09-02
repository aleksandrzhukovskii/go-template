package config

type Config struct {
	HttpGrpc   Server
	SqLite     Sqlite
	MySQL      MySQL
	Postgres   Postgres
	Mongo      Mongo
	Clickhouse Clickhouse

	Db     string `env:"DB,required"`
	Server string `env:"SERVER,required"`
}
