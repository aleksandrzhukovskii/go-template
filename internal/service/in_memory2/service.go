package in_memory2

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaswdr/faker/v2"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type Service struct {
	mu       sync.RWMutex
	products []model.Product
}

func New(_ config.Config) (model.DB, error) {
	return &Service{}, nil
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) findIndex(id string) (int, bool) {
	i := sort.Search(len(s.products), func(i int) bool {
		return s.products[i].ID >= id
	})
	if i < len(s.products) && s.products[i].ID == id {
		return i, true
	}
	return i, false
}

func (s *Service) Add(_ context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f := faker.NewWithSeed(rand.NewPCG(uint64(time.Now().Unix()), uint64(time.Now().UnixNano())))
	id := uuid.NewString()
	val := model.Product{
		ID:        id,
		Name:      f.Food().Fruit(),
		Price:     f.Float64(2, 1, 100),
		CreatedAt: uint32(time.Now().Unix()),
	}

	i, found := s.findIndex(id)
	if found {
		return "", errors.New("product already exists")
	}

	s.products = append(s.products, model.Product{})
	copy(s.products[i+1:], s.products[i:])
	s.products[i] = val
	return id, nil
}

func (s *Service) Update(_ context.Context, val model.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	i, found := s.findIndex(val.ID)
	if !found {
		return model.ErrorNoRowsUpdated
	}
	newVal := s.products[i]
	if val.Name != "" && val.Price != 0 {
		newVal.Price = val.Price
		newVal.Name = strings.Clone(val.Name)
	} else if val.Price != 0 {
		newVal.Price = val.Price
	} else if val.Name != "" {
		newVal.Name = strings.Clone(val.Name)
	} else {
		return model.ErrorNoUpdateParams
	}
	s.products[i] = newVal
	return nil
}

func (s *Service) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	i, found := s.findIndex(id)
	if !found {
		return model.ErrorNoRowsDeleted
	}
	s.products = append(s.products[:i], s.products[i+1:]...)
	return nil
}

func (s *Service) Get(_ context.Context, id string) (model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	i, found := s.findIndex(id)
	if !found {
		return model.Product{}, sql.ErrNoRows
	}
	return s.products[i], nil
}

func (s *Service) GetAll(_ context.Context) ([]model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.Product, len(s.products))
	copy(out, s.products)
	return out, nil
}
