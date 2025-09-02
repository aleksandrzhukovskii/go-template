package e2e_tests

import (
	"context"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
	pb "github.com/aleksandrzhukovskii/go-template/internal/service/grpc"
)

type GrpcSuite struct {
	suite.Suite
	conn   *grpc.ClientConn
	client pb.ProductServiceClient
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *GrpcSuite) SetupSuite() {
	var err error
	s.conn, err = grpc.NewClient("app-test:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.NoError(err)
	s.client = pb.NewProductServiceClient(s.conn)
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *GrpcSuite) TearDownSuite() {
	s.NoError(s.conn.Close())
	s.cancel()
}

func (s *GrpcSuite) Test_Main() {
	res, err := s.client.GetMain(s.ctx, &pb.Empty{})
	s.NoError(err)
	s.Contains(res.Info, "content-type: application/grpc")
	s.Contains(res.Info, "authority: app-test:8000")
}

func (s *GrpcSuite) Test_Product() {
	var productID string
	var productID2 string

	s.Run("Add Product", func() {
		resp, err := s.client.AddProduct(s.ctx, &pb.Empty{})
		s.NoError(err)
		s.NotEmpty(resp.Id, "Product ID should not be empty")
		productID = resp.Id
	})

	s.Run("Update Product Name", func() {
		resp, err := s.client.UpdateProduct(s.ctx, &pb.UpdateRequest{
			Id:   productID,
			Name: s.stringToPtr("Updated Product"),
		})
		s.NoError(err)
		s.Equal("Product updated", resp.Msg)
	})

	s.Run("Update Product Price", func() {
		resp, err := s.client.UpdateProduct(s.ctx, &pb.UpdateRequest{
			Id:    productID,
			Price: s.floatToPtr(99.99),
		})
		s.NoError(err)
		s.Equal("Product updated", resp.Msg)
	})

	s.Run("Update Product Nothing to update", func() {
		_, err := s.client.UpdateProduct(s.ctx, &pb.UpdateRequest{
			Id: productID,
		})
		s.Error(err)
		s.Contains(err.Error(), model.ErrorNoUpdateParams.Error())
	})

	s.Run("Update Product Not Exist", func() {
		_, err := s.client.UpdateProduct(s.ctx, &pb.UpdateRequest{
			Id:    "123",
			Price: s.floatToPtr(99.99),
		})
		s.Error(err)
		s.Contains(err.Error(), model.ErrorNoRowsUpdated.Error())
	})

	s.Run("Get Product", func() {
		res, err := s.client.GetProduct(s.ctx, &pb.GetProductRequest{Id: productID})
		s.NoError(err)
		s.Equal(productID, res.Id)
		s.Equal("Updated Product", res.Name)
		s.Equal(99.99, res.Price)
		s.NotEmpty(res.CreatedAt)
	})

	s.Run("Get All Products", func() {
		res, err := s.client.GetProducts(s.ctx, &pb.Empty{})
		s.NoError(err)
		s.Len(res.Items, 1)
		s.Equal(productID, res.Items[0].Id)
		s.Equal("Updated Product", res.Items[0].Name)
		s.Equal(99.99, res.Items[0].Price)
		s.NotEmpty(res.Items[0].CreatedAt)
	})

	s.Run("Add Product Fill More", func() {
		resp, err := s.client.AddProduct(s.ctx, &pb.Empty{})
		s.NoError(err)
		s.NotEmpty(resp.Id)
		productID2 = resp.Id
	})

	s.Run("Get All Products Check 2", func() {
		res, err := s.client.GetProducts(s.ctx, &pb.Empty{})
		s.NoError(err)
		s.Len(res.Items, 2)
	})

	s.Run("Delete Product", func() {
		res, err := s.client.DeleteProduct(s.ctx, &pb.DeleteRequest{Id: productID})
		s.NoError(err)
		s.Equal("Product deleted", res.Msg)
	})

	s.Run("Delete Product Second", func() {
		res, err := s.client.DeleteProduct(s.ctx, &pb.DeleteRequest{Id: productID2})
		s.NoError(err)
		s.Equal("Product deleted", res.Msg)
	})

	s.Run("Get Product Not Exist", func() {
		_, err := s.client.GetProduct(s.ctx, &pb.GetProductRequest{Id: productID})
		s.Error(err)
		s.Contains(err.Error(), "sql: no rows")
	})

	s.Run("Delete Product Not Exist", func() {
		_, err := s.client.DeleteProduct(s.ctx, &pb.DeleteRequest{Id: productID})
		s.Error(err)
		s.Contains(err.Error(), model.ErrorNoRowsDeleted.Error())
	})

	s.Run("Get All Products Empty", func() {
		res, err := s.client.GetProducts(s.ctx, &pb.Empty{})
		s.NoError(err)
		s.Empty(res.Items)
	})
}

func (s *GrpcSuite) stringToPtr(val string) *string {
	return &val
}

func (s *GrpcSuite) floatToPtr(val float64) *float64 {
	return &val
}
