package grpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

func (s Service) GetMain(ctx context.Context, _ *Empty) (*MainInfo, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.DataLoss, "missing metadata")
	}
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.DataLoss, "missing client address")
	}

	var response bytes.Buffer

	response.WriteString(fmt.Sprintf("Remote Addr: %s\nMetadata:\n", p.Addr.String()))
	for name, values := range md {
		for _, value := range values {
			response.WriteString(fmt.Sprintf("\t%s: %s\n", name, value))
		}
	}

	return &MainInfo{
		Info: response.String(),
	}, nil
}

func (s Service) AddProduct(ctx context.Context, _ *Empty) (*AddResponse, error) {
	id, err := s.db.Add(ctx)
	if err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
	}
	return &AddResponse{
		Id: id,
	}, nil
}

func (s Service) UpdateProduct(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error) {
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	price := float64(0)
	if req.Price != nil {
		price = *req.Price
	}
	prod, err := model.ParseProduct(req.Id, name, strconv.FormatFloat(price, 'f', -1, 64))
	if err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
	}
	err = s.db.Update(ctx, prod)
	if err != nil {
		code := codes.Aborted
		if errors.Is(err, model.ErrorNoRowsUpdated) {
			code = codes.InvalidArgument
		}
		return nil, status.Error(code, err.Error())
	}
	return &UpdateResponse{
		Msg: "Product updated",
	}, nil
}

func (s Service) DeleteProduct(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	if err := s.db.Delete(ctx, req.Id); err != nil {
		code := codes.Aborted
		if errors.Is(err, model.ErrorNoRowsDeleted) {
			code = codes.InvalidArgument
		}
		return nil, status.Error(code, err.Error())
	}
	return &DeleteResponse{
		Msg: "Product deleted",
	}, nil
}

func (s Service) GetProduct(ctx context.Context, req *GetProductRequest) (*Product, error) {
	val, err := s.db.Get(ctx, req.Id)
	if err != nil {
		code := codes.Aborted
		if errors.Is(err, model.ErrorNoRowsDeleted) {
			code = codes.InvalidArgument
		}
		return nil, status.Error(code, err.Error())
	}

	return mapProduct(val), nil
}
func (s Service) GetProducts(ctx context.Context, _ *Empty) (*Products, error) {
	val, err := s.db.GetAll(ctx)
	if err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
	}

	return &Products{
		Items: mapProducts(val),
	}, nil
}

func mapProduct(mod model.Product) *Product {
	return &Product{
		Id:        mod.ID,
		Name:      mod.Name,
		Price:     mod.Price,
		CreatedAt: mod.CreatedAt,
	}
}

func mapProducts(mod []model.Product) []*Product {
	products := make([]*Product, len(mod))
	for i := 0; i < len(mod); i++ {
		products[i] = mapProduct(mod[i])
	}
	return products
}
