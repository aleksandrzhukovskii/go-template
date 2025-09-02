package service

import (
	"context"
	"errors"
	"net"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
	"github.com/aleksandrzhukovskii/go-template/internal/service/clickhouse"
	"github.com/aleksandrzhukovskii/go-template/internal/service/fiber"
	"github.com/aleksandrzhukovskii/go-template/internal/service/gin"
	"github.com/aleksandrzhukovskii/go-template/internal/service/gorm"
	"github.com/aleksandrzhukovskii/go-template/internal/service/graphql"
	"github.com/aleksandrzhukovskii/go-template/internal/service/grpc"
	"github.com/aleksandrzhukovskii/go-template/internal/service/in_memory"
	"github.com/aleksandrzhukovskii/go-template/internal/service/in_memory2"
	"github.com/aleksandrzhukovskii/go-template/internal/service/mongo"
	"github.com/aleksandrzhukovskii/go-template/internal/service/mysql"
	"github.com/aleksandrzhukovskii/go-template/internal/service/net_http"
	"github.com/aleksandrzhukovskii/go-template/internal/service/postgres"
	"github.com/aleksandrzhukovskii/go-template/internal/service/sqlite"
	"github.com/aleksandrzhukovskii/go-template/internal/service/yaml_to_code"
)

type Services struct {
	server model.Server
	db     model.DB
	lis    net.Listener
}

func New(cfg config.Config) (*Services, error) {
	return newServices(cfg, nil)
}

func NewWithListener(cfg config.Config, listener net.Listener) (*Services, error) {
	return newServices(cfg, listener)
}

func newServices(cfg config.Config, lis net.Listener) (*Services, error) {
	ret := new(Services)
	var err error

	dbNewFunc, ok := dbNew[cfg.Db]
	if !ok {
		return nil, errors.New("database driver not found")
	}

	serverNewFunc, ok := serverNew[cfg.Server]
	if !ok {
		return nil, errors.New("server type not found")
	}

	ret.db, err = dbNewFunc(cfg)
	if err != nil {
		return nil, err
	}

	/*if err = migrator.Migrate(cfg); err != nil {
		return nil, err
	}*/

	if lis == nil {
		addr := cfg.HttpGrpc.IP + ":" + cfg.HttpGrpc.Port
		lis, err = net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
	}
	ret.lis = lis

	ret.server, err = serverNewFunc(ret.db, lis)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

var dbNew = map[string]func(cfg config.Config) (model.DB, error){
	"sqlite":        sqlite.New,
	"mysql":         mysql.New,
	"postgres":      postgres.New,
	"mongo":         mongo.New,
	"clickhouse":    clickhouse.New,
	"in_memory":     in_memory.New,
	"in_memory2":    in_memory2.New,
	"gorm_postgres": gorm.New,
	"gorm_mysql":    gorm.New,
	"gorm_sqlite":   gorm.New,
}

var serverNew = map[string]func(db model.DB, lis net.Listener) (model.Server, error){
	"net_http":     net_http.New,
	"gin":          gin.New,
	"fiber":        fiber.New,
	"grpc":         grpc.New,
	"graphql":      graphql.New,
	"yaml_to_code": yaml_to_code.New,
}

func (s *Services) Start(ctx context.Context) error {
	if err := s.db.Start(); err != nil {
		return err
	}
	if err := s.server.Start(ctx); err != nil {
		return err
	}
	return s.lis.Close()
}
