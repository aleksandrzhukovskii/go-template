package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jaswdr/faker/v2"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type Service struct {
	dial gorm.Dialector
	db   *gorm.DB
}

func New(cfg config.Config) (model.DB, error) {
	switch cfg.Db {
	case "gorm_postgres":
		return &Service{
			dial: postgres.Open(cfg.Postgres.GormDNS()),
		}, nil
	case "gorm_mysql":
		return &Service{
			dial: mysql.Open(cfg.MySQL.DSN()),
		}, nil
	case "gorm_sqlite":
		return &Service{
			dial: sqlite.Dialector{
				DriverName: "sqlite",
				DSN:        "file:" + cfg.SqLite.Path + "?cache=shared&_fk=1",
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Db)
	}
}

func (s *Service) Start() error {
	db, err := gorm.Open(s.dial, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}
	s.db = db

	return s.migrate()
}

func (s *Service) migrate() error {
	return s.db.AutoMigrate(&model.Product{})
}

func (s *Service) Add(ctx context.Context) (string, error) {
	f := faker.NewWithSeed(rand.NewPCG(uint64(time.Now().Unix()), uint64(time.Now().UnixNano())))
	id := uuid.NewString()

	product := model.Product{
		ID:        id,
		Name:      f.Food().Fruit(),
		Price:     f.Float64(2, 1, 100),
		CreatedAt: uint32(time.Now().Unix()),
	}

	if err := s.db.WithContext(ctx).Create(&product).Error; err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) Update(ctx context.Context, val model.Product) error {
	if val.Name == "" && val.Price == 0 {
		return model.ErrorNoUpdateParams
	}

	res := s.db.WithContext(ctx).Model(&model.Product{}).Where("id = ?", val.ID)

	updates := map[string]interface{}{}
	if val.Name != "" {
		updates["name"] = val.Name
	}
	if val.Price != 0 {
		updates["price"] = val.Price
	}

	tx := res.WithContext(ctx).Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return model.ErrorNoRowsUpdated
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	tx := s.db.WithContext(ctx).Delete(&model.Product{}, "id = ?", id)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return model.ErrorNoRowsDeleted
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (model.Product, error) {
	var ret model.Product
	if err := s.db.WithContext(ctx).First(&ret, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Product{}, sql.ErrNoRows
		}
		return model.Product{}, err
	}
	return ret, nil
}

func (s *Service) GetAll(ctx context.Context) ([]model.Product, error) {
	var ret []model.Product
	if err := s.db.WithContext(ctx).Find(&ret).Error; err != nil {
		return nil, err
	}
	return ret, nil
}
