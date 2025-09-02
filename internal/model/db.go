package model

import (
	"context"
	"errors"
	"strconv"
)

const TableName = "products"

type Product struct {
	ID        string  `json:"id" bson:"id" db:"id"`
	Name      string  `json:"name,omitempty" bson:"name" db:"name"`
	Price     float64 `json:"price,omitempty" bson:"price" db:"price"`
	CreatedAt uint32  `json:"created_at,omitempty" bson:"created_at" db:"created_at"`
}

func ParseProduct(id string, name string, priceStr string) (Product, error) {
	if id == "" {
		return Product{}, errors.New("invalid id")
	}
	ret := Product{
		ID: id,
	}
	price, err := strconv.ParseFloat(priceStr, 64)
	if err == nil {
		ret.Price = price
	}
	if name != "" {
		ret.Name = name
	}
	if ret.Name == "" && ret.Price == 0 {
		return Product{}, ErrorNoUpdateParams
	}
	return ret, nil
}

type DB interface {
	Add(ctx context.Context) (string, error)
	Update(ctx context.Context, val Product) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (Product, error)
	GetAll(ctx context.Context) ([]Product, error)
	Start() error
}
