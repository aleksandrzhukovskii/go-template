package in_memory

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/jaswdr/faker/v2"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type Service struct {
	db *memdb.MemDB
}

func New(_ config.Config) (model.DB, error) {
	var schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.TableName: {
				Name: model.TableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
				},
			},
		},
	}
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}

	return &Service{
		db: db,
	}, nil
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Add(_ context.Context) (str string, err error) {
	tx := s.db.Txn(true)
	defer func() {
		if err != nil {
			tx.Abort()
		} else {
			tx.Commit()
		}
	}()
	f := faker.NewWithSeed(rand.NewPCG(uint64(time.Now().Unix()), uint64(time.Now().UnixNano())))
	id := uuid.NewString()

	err = tx.Insert(model.TableName, &model.Product{
		ID:        id,
		Name:      f.Food().Fruit(),
		Price:     f.Float64(2, 1, 100),
		CreatedAt: uint32(time.Now().Unix()),
	})
	if err != nil {
		return "", err
	}
	return id, err
}

func (s *Service) Update(_ context.Context, val model.Product) (err error) {
	tx := s.db.Txn(true)
	defer func() {
		if err != nil {
			tx.Abort()
		} else {
			tx.Commit()
		}
	}()
	raw, err := tx.First(model.TableName, "id", val.ID)
	if err != nil {
		return err
	}
	if raw == nil {
		return model.ErrorNoRowsUpdated
	}
	if val.Name != "" && val.Price != 0 {
		raw.(*model.Product).Price = val.Price
		raw.(*model.Product).Name = strings.Clone(val.Name)
	} else if val.Price != 0 {
		raw.(*model.Product).Price = val.Price
	} else if val.Name != "" {
		raw.(*model.Product).Name = strings.Clone(val.Name)
	} else {
		return model.ErrorNoUpdateParams
	}

	return nil
}

func (s *Service) Delete(_ context.Context, id string) (err error) {
	tx := s.db.Txn(true)
	defer func() {
		if err != nil {
			tx.Abort()
		} else {
			tx.Commit()
		}
	}()
	err = tx.Delete(model.TableName, &model.Product{ID: id})
	if err != nil {
		if errors.Is(err, memdb.ErrNotFound) {
			return model.ErrorNoRowsDeleted
		}
		return err
	}
	return nil
}

func (s *Service) Get(_ context.Context, id string) (_ model.Product, err error) {
	tx := s.db.Txn(false)
	defer func() {
		if err != nil {
			tx.Abort()
		} else {
			tx.Commit()
		}
	}()
	raw, err := tx.First(model.TableName, "id", id)
	if err != nil {
		return model.Product{}, err
	}
	if raw == nil {
		return model.Product{}, sql.ErrNoRows
	}
	return *raw.(*model.Product), nil
}

func (s *Service) GetAll(_ context.Context) (_ []model.Product, err error) {
	tx := s.db.Txn(false)
	defer func() {
		if err != nil {
			tx.Abort()
		} else {
			tx.Commit()
		}
	}()
	it, err := tx.Get(model.TableName, "id")
	if err != nil {
		return nil, err
	}
	var ret []model.Product
	for obj := it.Next(); obj != nil; obj = it.Next() {
		ret = append(ret, *obj.(*model.Product))
	}
	return ret, nil
}
