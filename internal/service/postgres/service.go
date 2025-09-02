package postgres

import (
	"context"
	"database/sql"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jaswdr/faker/v2"
	_ "github.com/lib/pq"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type Service struct {
	dns string
	db  *sql.DB
}

func New(cfg config.Config) (model.DB, error) {
	return &Service{
		dns: cfg.Postgres.DSN(),
	}, nil
}

func (s *Service) Start() error {
	db, err := sql.Open("postgres", s.dns)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return err
	}
	s.db = db
	return s.migrate()
}

// You'd better use https://hub.docker.com/r/migrate/migrate or so in real project
func (s *Service) migrate() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS products (
			id TEXT PRIMARY KEY,
			name TEXT,
			price DOUBLE PRECISION,
			created_at BIGINT
		);`)
	return err
}

func (s *Service) Add(ctx context.Context) (string, error) {
	f := faker.NewWithSeed(rand.NewPCG(uint64(time.Now().Unix()), uint64(time.Now().UnixNano())))
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx, "INSERT INTO products(id, name, price, created_at) values ($1,$2,$3,$4)",
		id, f.Food().Fruit(), f.Float64(2, 1, 100), uint32(time.Now().Unix()))
	return id, err
}

func (s *Service) Update(ctx context.Context, val model.Product) error {
	var res sql.Result
	var err error
	if val.Name != "" && val.Price != 0 {
		res, err = s.db.ExecContext(ctx, "UPDATE products SET name=$1, price=$2 WHERE id=$3", val.Name, val.Price, val.ID)
	} else if val.Price != 0 {
		res, err = s.db.ExecContext(ctx, "UPDATE products SET price=$1 WHERE id=$2", val.Price, val.ID)
	} else if val.Name != "" {
		res, err = s.db.ExecContext(ctx, "UPDATE products SET name=$1 WHERE id=$2", val.Name, val.ID)
	} else {
		return model.ErrorNoUpdateParams
	}

	if err != nil {
		return err
	}
	if cnt, _ := res.RowsAffected(); cnt == 0 {
		return model.ErrorNoRowsUpdated
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM products WHERE id=$1", id)
	if err != nil {
		return err
	}
	if cnt, _ := res.RowsAffected(); cnt == 0 {
		return model.ErrorNoRowsDeleted
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (model.Product, error) {
	row := s.db.QueryRowContext(ctx, "SELECT * FROM products WHERE id=$1", id)
	if err := row.Err(); err != nil {
		return model.Product{}, err
	}
	var ret model.Product
	err := row.Scan(&ret.ID, &ret.Name, &ret.Price, &ret.CreatedAt)
	if err != nil {
		return model.Product{}, err
	}
	return ret, nil
}

func (s *Service) GetAll(ctx context.Context) ([]model.Product, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * FROM products")
	if err != nil {
		return nil, err
	}
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
