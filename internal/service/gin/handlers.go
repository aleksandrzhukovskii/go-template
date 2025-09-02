package gin

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

func (s *Service) Main(ctx *gin.Context) {
	body, err := ctx.GetRawData()
	if err != nil {
		s.sendError(ctx, http.StatusInternalServerError, err)
	}

	var response bytes.Buffer

	response.WriteString(fmt.Sprintf("Method: %s\nURL: %s\nRemote Addr: %s\nHeaders:\n", ctx.Request.Method,
		ctx.Request.URL.String(), ctx.Request.RemoteAddr))
	for name, values := range ctx.Request.Header {
		for _, value := range values {
			response.WriteString(fmt.Sprintf("\t%s: %s\n", name, value))
		}
	}

	if len(body) > 0 {
		response.WriteString(fmt.Sprintf("Body: %s\n", string(body)))
	}

	response.WriteString(fmt.Sprintf("Swagger: http://%s/swagger\n", s.server.Addr))

	ctx.Data(http.StatusOK, "text/html; charset=utf-8", response.Bytes())
}

func (s *Service) AddProduct(ctx *gin.Context) {
	id, err := s.db.Add(ctx)
	if err != nil {
		s.sendError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, model.Product{ID: id})
}

func (s *Service) UpdateProduct(ctx *gin.Context) {
	id, _ := ctx.GetPostForm("id")
	name, _ := ctx.GetPostForm("name")
	price, _ := ctx.GetPostForm("price")
	prod, err := model.ParseProduct(id, name, price)
	if err != nil {
		s.sendError(ctx, http.StatusBadRequest, err)
		return
	}
	err = s.db.Update(ctx, prod)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, model.ErrorNoRowsUpdated) {
			status = http.StatusBadRequest
		}
		s.sendError(ctx, status, err)
		return
	}
	ctx.JSON(http.StatusOK, struct {
		Msg string `json:"msg"`
	}{
		Msg: "Product updated",
	})
}

func (s *Service) DeleteProduct(ctx *gin.Context) {
	id, _ := ctx.GetQuery("id")
	if err := s.db.Delete(ctx, id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, model.ErrorNoRowsDeleted) {
			status = http.StatusBadRequest
		}
		s.sendError(ctx, status, err)
		return
	}
	ctx.JSON(http.StatusOK, struct {
		Msg string `json:"msg"`
	}{
		Msg: "Product deleted",
	})
}

func (s *Service) GetProduct(ctx *gin.Context) {
	id, _ := ctx.GetQuery("id")
	val, err := s.db.Get(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusBadRequest
		}
		s.sendError(ctx, status, err)
		return
	}

	ctx.JSON(http.StatusOK, val)
}

func (s *Service) GetProducts(ctx *gin.Context) {
	val, err := s.db.GetAll(ctx)
	if err != nil {
		s.sendError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, val)
}

func (s *Service) sendError(ctx *gin.Context, status int, err error) {
	ctx.JSON(status, gin.H{
		"error": err.Error(),
	})
}
