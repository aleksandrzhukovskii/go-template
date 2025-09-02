package mongo

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jaswdr/faker/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type Service struct {
	dns    string
	dbName string
	db     *mongo.Client
	c      *mongo.Collection
}

func New(cfg config.Config) (model.DB, error) {
	return &Service{
		dns:    cfg.Mongo.DSN(),
		dbName: cfg.Mongo.Database,
	}, nil
}

func (s *Service) Start() error {
	client, err := mongo.Connect(options.Client().ApplyURI(s.dns))
	if err != nil {
		return err
	}
	s.c = client.Database(s.dbName).Collection(model.TableName)
	s.db = client
	return nil
}

func (s *Service) Add(ctx context.Context) (string, error) {
	f := faker.NewWithSeed(rand.NewPCG(uint64(time.Now().Unix()), uint64(time.Now().UnixNano())))
	id := uuid.NewString()
	_, err := s.c.InsertOne(ctx, model.Product{
		ID:        id,
		Name:      f.Food().Fruit(),
		Price:     f.Float64(2, 1, 100),
		CreatedAt: uint32(time.Now().Unix()),
	})

	return id, err
}

func (s *Service) Update(ctx context.Context, val model.Product) error {
	update := bson.M{}
	if val.Name != "" {
		update["name"] = val.Name
	}
	if val.Price != 0 {
		update["price"] = val.Price
	}
	if len(update) == 0 {
		return model.ErrorNoUpdateParams
	}

	res, err := s.c.UpdateOne(ctx, bson.M{"id": val.ID}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return model.ErrorNoRowsUpdated
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	res, err := s.c.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return model.ErrorNoRowsDeleted
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (model.Product, error) {
	var result model.Product
	err := s.c.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Product{}, sql.ErrNoRows
		}
		return model.Product{}, err
	}
	return result, nil
}

func (s *Service) GetAll(ctx context.Context) ([]model.Product, error) {
	cursor, err := s.c.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []model.Product
	for cursor.Next(ctx) {
		var p model.Product
		if err = cursor.Decode(&p); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, nil
}
