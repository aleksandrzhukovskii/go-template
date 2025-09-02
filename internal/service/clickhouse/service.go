package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/jaswdr/faker/v2"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type Service struct {
	opt *clickhouse.Options
	db  clickhouse.Conn
}

func New(cfg config.Config) (model.DB, error) {
	return &Service{
		opt: &clickhouse.Options{
			Addr: []string{cfg.Clickhouse.Host + ":" + strconv.Itoa(cfg.Clickhouse.Port)},
			Settings: clickhouse.Settings{
				"max_execution_time": 60,
				"mutations_sync":     1,
			},
			Auth: clickhouse.Auth{
				Database: cfg.Clickhouse.Database,
				Username: cfg.Clickhouse.User,
				Password: cfg.Clickhouse.Password,
			},
			Compression: &clickhouse.Compression{
				Method: clickhouse.CompressionLZ4,
			},
			DialTimeout:          time.Second * 30,
			MaxOpenConns:         5,
			MaxIdleConns:         5,
			ConnMaxLifetime:      time.Duration(10) * time.Minute,
			ConnOpenStrategy:     clickhouse.ConnOpenInOrder,
			BlockBufferSize:      10,
			MaxCompressionBuffer: 10240,
		},
	}, nil
}

func (s *Service) Start() error {
	conn, err := clickhouse.Open(s.opt)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	if err = conn.Ping(ctx); err != nil {
		return err
	}

	s.db = conn
	return s.migrate()
}

func (s *Service) migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	return s.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS products (
			id String,
			name String,
			price Float64,
			created_at UInt32
		) ENGINE = MergeTree()
		ORDER BY id
	`)
}

func (s *Service) Add(ctx context.Context) (string, error) {
	f := faker.NewWithSeed(rand.NewPCG(uint64(time.Now().Unix()), uint64(time.Now().UnixNano())))
	id := uuid.NewString()
	return id, s.db.Exec(ctx, "INSERT INTO products (id, name, price, created_at) VALUES (?, ?, ?, ?)",
		id, f.Food().Fruit(), f.Float64(2, 1, 100), uint32(time.Now().Unix()))
}

func (s *Service) Update(ctx context.Context, val model.Product) error {
	var exists uint8
	err := s.db.QueryRow(ctx, "SELECT 1 FROM products WHERE id = ? LIMIT 1", val.ID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrorNoRowsUpdated
		}
		return err
	}

	if val.Name != "" && val.Price != 0 {
		err = s.db.Exec(ctx, "ALTER TABLE products UPDATE name = ?, price = ? WHERE id = ?", val.Name, val.Price, val.ID)
	} else if val.Price != 0 {
		err = s.db.Exec(ctx, "ALTER TABLE products UPDATE price = ? WHERE id = ?", val.Price, val.ID)
	} else if val.Name != "" {
		err = s.db.Exec(ctx, "ALTER TABLE products UPDATE name = ? WHERE id = ?", val.Name, val.ID)
	} else {
		return model.ErrorNoUpdateParams
	}
	return err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	var exists uint8
	err := s.db.QueryRow(ctx, "SELECT 1 FROM products WHERE id = ? LIMIT 1", id).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrorNoRowsDeleted
		}
		return err
	}

	return s.db.Exec(ctx, "ALTER TABLE products DELETE WHERE id = ?", id)
}

func (s *Service) Get(ctx context.Context, id string) (model.Product, error) {
	var ret model.Product
	row := s.db.QueryRow(ctx, "SELECT id, name, price, created_at FROM products WHERE id = ? LIMIT 1", id)
	if err := row.Scan(&ret.ID, &ret.Name, &ret.Price, &ret.CreatedAt); err != nil {
		return model.Product{}, err
	}
	return ret, nil
}

func (s *Service) GetAll(ctx context.Context) ([]model.Product, error) {
	rows, err := s.db.Query(ctx, "SELECT id, name, price, created_at FROM products")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ret []model.Product
	for rows.Next() {
		var elem model.Product
		if err = rows.Scan(&elem.ID, &elem.Name, &elem.Price, &elem.CreatedAt); err != nil {
			return nil, err
		}
		ret = append(ret, elem)
	}
	return ret, nil
}
