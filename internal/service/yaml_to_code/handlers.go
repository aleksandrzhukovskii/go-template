package yaml_to_code

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

func (s *Service) GetMain(ctx context.Context, _ GetMainRequestObject) (GetMainResponseObject, error) {
	r := ctx.Value(reqKey).(*http.Request)
	var body []byte
	var err error
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return GetMain500TextResponse(err.Error()), nil
		}
	}

	var response bytes.Buffer

	response.WriteString(fmt.Sprintf("Method: %s\nURL: %s\nRemote Addr: %s\nHeaders:\n", r.Method,
		r.URL.String(), r.RemoteAddr))
	for name, values := range r.Header {
		for _, value := range values {
			response.WriteString(fmt.Sprintf("\t%s: %s\n", name, value))
		}
	}

	if len(body) > 0 {
		response.WriteString(fmt.Sprintf("Body: %s\n", string(body)))
	}

	response.WriteString(fmt.Sprintf("Swagger: http://%s/swagger\n", s.server.Addr))

	return GetMain200TextResponse(response.String()), nil
}

func (s *Service) AddProduct(ctx context.Context, _ AddProductRequestObject) (AddProductResponseObject, error) {
	id, err := s.db.Add(ctx)
	if err != nil {
		return AddProduct500JSONResponse{
			DbIssueJSONResponse{
				Error: err.Error(),
			},
		}, nil
	}
	return AddProduct200JSONResponse{
		Id: id,
	}, nil
}

func (s *Service) DeleteProduct(ctx context.Context, request DeleteProductRequestObject) (DeleteProductResponseObject, error) {
	if err := s.db.Delete(ctx, request.Params.Id); err != nil {
		if errors.Is(err, model.ErrorNoRowsDeleted) {
			return DeleteProduct400JSONResponse{
				Error: err.Error(),
			}, nil
		}
		return DeleteProduct500JSONResponse{
			DbIssueJSONResponse{
				Error: err.Error(),
			},
		}, nil
	}
	return DeleteProduct200JSONResponse{
		Msg: "Product deleted",
	}, nil
}

func (s *Service) GetProduct(ctx context.Context, request GetProductRequestObject) (GetProductResponseObject, error) {
	val, err := s.db.Get(ctx, request.Params.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetProduct400JSONResponse{
				NoRowsJSONResponse{
					Error: err.Error(),
				},
			}, nil
		}
		return GetProduct500JSONResponse{
			DbIssueJSONResponse{
				Error: err.Error(),
			},
		}, nil
	}
	return GetProduct200JSONResponse(val), nil
}

func (s *Service) GetProducts(ctx context.Context, _ GetProductsRequestObject) (GetProductsResponseObject, error) {
	val, err := s.db.GetAll(ctx)
	if err != nil {
		return GetProducts500JSONResponse{
			DbIssueJSONResponse{
				Error: err.Error(),
			},
		}, nil
	}
	return GetProducts200JSONResponse(val), nil
}

func (s *Service) UpdateProduct(ctx context.Context, request UpdateProductRequestObject) (UpdateProductResponseObject, error) {
	if request.Body.Name == nil && request.Body.Price == nil {
		return UpdateProduct400JSONResponse{NoUpdateJSONResponse{
			Error: model.ErrorNoUpdateParams.Error(),
		}}, nil
	}
	prod := model.Product{
		ID: request.Body.Id,
	}
	if request.Body.Name != nil {
		prod.Name = *request.Body.Name
	}
	if request.Body.Price != nil {
		prod.Price = *request.Body.Price
	}

	if err := s.db.Update(ctx, prod); err != nil {
		if errors.Is(err, model.ErrorNoRowsUpdated) {
			return UpdateProduct400JSONResponse{
				NoUpdateJSONResponse{
					Error: err.Error(),
				},
			}, nil
		}
		return UpdateProduct500JSONResponse{
			DbIssueJSONResponse{
				Error: err.Error(),
			},
		}, nil
	}
	return UpdateProduct200JSONResponse{
		Msg: "Product updated",
	}, nil
}
