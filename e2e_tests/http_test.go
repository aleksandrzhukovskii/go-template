package e2e_tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
	"github.com/stretchr/testify/suite"
)

type HTTPSuite struct {
	suite.Suite
}

func (s *HTTPSuite) Test_Main() {
	res, err := http.Get("http://app-test:8000")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	s.NoError(err)
	s.True(bytes.HasPrefix(data, []byte("Method: GET\nURL: /")))
}

func (s *HTTPSuite) Test_Swagger() {
	res, err := http.Get("http://app-test:8000/swagger/")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	res.Body.Close()
}

func (s *HTTPSuite) Test_SwaggerConfig() {
	res, err := http.Get("http://app-test:8000/swagger.yaml")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	res.Body.Close()
}

func (s *HTTPSuite) Test_Product() {
	var productID string
	var productID2 string
	tests := []struct {
		name       string
		method     string
		path       string
		params     map[string]any
		wantStatus int
		check      func(body io.Reader)
	}{
		{
			name:       "Add Product",
			method:     http.MethodPost,
			path:       "/add",
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.NotEmpty(result["id"], "Product ID should not be empty")
				productID = result["id"].(string)
			},
		},
		{
			name:       "Update Product Name",
			method:     http.MethodPut,
			path:       "/update",
			params:     map[string]any{"id": &productID, "name": s.stringToPtr("Updated Product")},
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal("Product updated", result["msg"], "Update should be successful")
			},
		},
		{
			name:       "Update Product Price",
			method:     http.MethodPut,
			path:       "/update",
			params:     map[string]any{"id": &productID, "price": 99.99},
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal("Product updated", result["msg"], "Update should be successful")
			},
		},
		{
			name:       "Update Product Nothing to update",
			method:     http.MethodPut,
			path:       "/update",
			params:     map[string]any{"id": &productID},
			wantStatus: http.StatusBadRequest,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal(model.ErrorNoUpdateParams.Error(), result["error"], "Update should fail")
			},
		},
		{
			name:       "Update Product Not Exist",
			method:     http.MethodPut,
			path:       "/update",
			params:     map[string]any{"id": s.stringToPtr("123"), "price": 99.99},
			wantStatus: http.StatusBadRequest,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal(model.ErrorNoRowsUpdated.Error(), result["error"], "Update should fail")
			},
		},
		{
			name:       "Get Product",
			method:     http.MethodGet,
			path:       "/get",
			params:     map[string]any{"id": &productID},
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal(productID, result["id"], "Retrieved product ID should match")
				s.Equal("Updated Product", result["name"], "Retrieved product name should match")
				s.Equal(99.99, result["price"], "Retrieved product price should match")
				s.NotEmpty(result["created_at"], "Retrieved product create time should be filled")
			},
		},
		{
			name:       "Get All Products",
			method:     http.MethodGet,
			path:       "/get_all",
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getSlice(body)
				s.Len(result, 1, "Result should contain 1 product")
				s.Equal(productID, result[0]["id"], "Retrieved product ID should match")
				s.Equal("Updated Product", result[0]["name"], "Retrieved product name should match")
				s.Equal(99.99, result[0]["price"], "Retrieved product price should match")
				s.NotEmpty(result[0]["created_at"], "Retrieved product create time should be filled")
			},
		},
		{
			name:       "Add Product Fill More",
			method:     http.MethodPost,
			path:       "/add",
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.NotEmpty(result["id"], "Product ID should not be empty")
				productID2 = result["id"].(string)
			},
		},
		{
			name:       "Get All Products Check 2",
			method:     http.MethodGet,
			path:       "/get_all",
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getSlice(body)
				s.Len(result, 2, "Result should contain 2 products")
			},
		},
		{
			name:       "Delete Product",
			method:     http.MethodDelete,
			path:       "/delete",
			params:     map[string]any{"id": &productID},
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal("Product deleted", result["msg"], "Delete should be successful")
			},
		},
		{
			name:       "Delete Product Second",
			method:     http.MethodDelete,
			path:       "/delete",
			params:     map[string]any{"id": &productID2},
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal("Product deleted", result["msg"], "Delete should be successful")
			},
		},
		{
			name:       "Get Product Not Exist",
			method:     http.MethodGet,
			path:       "/get",
			params:     map[string]any{"id": &productID},
			wantStatus: http.StatusBadRequest,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal(sql.ErrNoRows.Error(), result["error"], "Get should fail")
			},
		},
		{
			name:       "Delete Product Not Exist",
			method:     http.MethodDelete,
			path:       "/delete",
			params:     map[string]any{"id": &productID},
			wantStatus: http.StatusBadRequest,
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Equal(model.ErrorNoRowsDeleted.Error(), result["error"], "Delete should fail")
			},
		},
		{
			name:       "Get All Products Empty",
			method:     http.MethodGet,
			path:       "/get_all",
			wantStatus: http.StatusOK,
			check: func(body io.Reader) {
				result := s.getSlice(body)
				s.Empty(result, "Result should be empty")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var req *http.Request
			var err error

			params := url.Values{}
			for k, v := range tc.params {
				if str, ok := v.(*string); ok {
					params.Set(k, *str)
				} else {
					params.Set(k, fmt.Sprintf("%v", v))
				}
			}

			if tc.method == http.MethodGet || tc.method == http.MethodDelete {
				req, err = http.NewRequest(tc.method, "http://app-test:8000"+tc.path+"?"+params.Encode(), nil)
			} else {
				req, err = http.NewRequest(tc.method, "http://app-test:8000"+tc.path, bytes.NewBufferString(params.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			s.NoError(err)

			resp, err := http.DefaultClient.Do(req)
			s.NoError(err)
			s.Equal(tc.wantStatus, resp.StatusCode)
			defer resp.Body.Close()
			if tc.check != nil {
				tc.check(resp.Body)
			}
		})
	}
}

func (s *HTTPSuite) getMap(body io.Reader) map[string]any {
	result := make(map[string]any)
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		s.FailNow(err.Error())
	}
	return result
}

func (s *HTTPSuite) getSlice(body io.Reader) []map[string]any {
	var result []map[string]any
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		s.FailNow(err.Error())
	}
	return result
}

func (s *HTTPSuite) stringToPtr(str string) *string {
	return &str
}
