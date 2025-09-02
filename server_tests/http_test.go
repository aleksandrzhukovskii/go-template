package server_tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/test/bufconn"
	_ "modernc.org/sqlite"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type HTTPSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	path   string

	db     string
	server string
	lis    *bufconn.Listener
	client *http.Client
}

func (s *HTTPSuite) SetupSuite() {
	s.path = os.TempDir() + "/db"
	if _, err := os.Stat(s.path); err == nil {
		db, err := sql.Open("sqlite", s.path)
		s.NoError(err)
		_, err = db.Exec("DELETE FROM products WHERE 1=1")
		s.NoError(err)
		s.NoError(db.Close())
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wg.Add(1)

	if err := os.Setenv("DB", s.db); err != nil {
		s.FailNow(err.Error())
	}
	if err := os.Setenv("SERVER", s.server); err != nil {
		s.FailNow(err.Error())
	}
	if err := os.Setenv("IP", "127.0.0.1"); err != nil {
		s.FailNow(err.Error())
	}
	if err := os.Setenv("PORT", "123"); err != nil {
		s.FailNow(err.Error())
	}

	s.lis = bufconn.Listen(1024 * 1024)
	go func() {
		MainAnalogue(s.ctx, s.lis)
		s.wg.Done()
	}()
	time.Sleep(time.Second)

	s.client = &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return s.lis.Dial()
			},
		},
	}
}

func (s *HTTPSuite) TearDownSuite() {
	s.cancel()
	s.wg.Wait()

	_ = os.Remove(s.path)
}

func (s *HTTPSuite) Test_Main() {
	res, err := s.client.Get("http://127.0.0.1:8000")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	s.NoError(err)
	s.True(bytes.HasPrefix(data, []byte("Method: GET\nURL: /")))
}

func (s *HTTPSuite) Test_Swagger() {
	res, err := s.client.Get("http://127.0.0.1:8000/swagger/")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	res.Body.Close()
}

func (s *HTTPSuite) Test_SwaggerConfig() {
	res, err := s.client.Get("http://127.0.0.1:8000/swagger.yaml")
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
				req, err = http.NewRequest(tc.method, "http://127.0.0.1:8000"+tc.path+"?"+params.Encode(), nil)
			} else {
				req, err = http.NewRequest(tc.method, "http://127.0.0.1:8000"+tc.path, bytes.NewBufferString(params.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			s.NoError(err)

			resp, err := s.client.Do(req)
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

func TestNetHttpSqlite(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "sqlite",
		server: "net_http",
	})
}

func TestGinSqlite(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "sqlite",
		server: "gin",
	})
}

func TestFiberSqlite(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "sqlite",
		server: "fiber",
	})
}

func TestYamlSqlite(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "sqlite",
		server: "yaml_to_code",
	})
}

func TestNetHttpMemory(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory",
		server: "net_http",
	})
}

func TestGinMemory(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory",
		server: "gin",
	})
}

func TestFiberMemory(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory",
		server: "fiber",
	})
}

func TestYamlMemory(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory",
		server: "yaml_to_code",
	})
}

func TestNetHttpMemory2(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory2",
		server: "net_http",
	})
}

func TestGinMemory2(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory2",
		server: "gin",
	})
}

func TestFiberMemory2(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory2",
		server: "fiber",
	})
}

func TestYamlMemory2(t *testing.T) {
	suite.Run(t, &HTTPSuite{
		db:     "in_memory2",
		server: "yaml_to_code",
	})
}
