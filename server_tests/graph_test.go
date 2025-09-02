package server_tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/test/bufconn"
	_ "modernc.org/sqlite"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type GraphSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	path   string

	db     string
	lis    *bufconn.Listener
	client *http.Client
}

func (s *GraphSuite) SetupSuite() {
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
	if err := os.Setenv("SERVER", "graphql"); err != nil {
		s.FailNow(err.Error())
	}
	if err := os.Setenv("IP", "127.0.0.1"); err != nil {
		s.FailNow(err.Error())
	}
	if err := os.Setenv("PORT", "8000"); err != nil {
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

func (s *GraphSuite) TearDownSuite() {
	s.cancel()
	s.wg.Wait()

	_ = os.Remove(s.path)
}

func (s *GraphSuite) Test_QueryPlayground() {
	res, err := s.client.Get("http://127.0.0.1:8000/query_playground")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	res.Body.Close()
}

func (s *GraphSuite) Test_SubscriptionPlayground() {
	res, err := s.client.Get("http://127.0.0.1:8000/subscription_playground")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	res.Body.Close()
}

func (s *GraphSuite) Test_Main() {
	res, err := s.client.Post("http://127.0.0.1:8000/query", "application/json",
		strings.NewReader(`{"query":"{ main }"}`))
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	s.NoError(err)
	data1, err := jsonparser.GetString(data, "data", "main")
	s.NoError(err)
	s.True(strings.HasPrefix(data1, "Method: POST\nURL: /"))
}

func (s *GraphSuite) Test_Subscription() {
	u, err := url.Parse("ws://127.0.0.1:8000/subscription")
	s.NoError(err)

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Subprotocols:     []string{"graphql-transport-ws"},
		NetDial:          func(_, _ string) (net.Conn, error) { return s.lis.Dial() },
	}
	c, _, err := dialer.Dial(u.String(), nil)
	s.NoError(err)
	defer c.Close()

	initMsg := map[string]interface{}{
		"type": "connection_init",
	}
	err = c.WriteJSON(initMsg)
	s.NoError(err)

	_, recMsg, err := c.ReadMessage()
	s.NoError(err)
	var ack map[string]interface{}
	err = json.Unmarshal(recMsg, &ack)
	s.NoError(err)
	s.Equal("connection_ack", ack["type"])

	subscriptionQuery := map[string]interface{}{
		"id":   uuid.NewString(),
		"type": "subscribe",
		"payload": map[string]string{
			"query": "subscription { time }",
		},
	}
	err = c.WriteJSON(subscriptionQuery)
	s.NoError(err)
	timeout := time.After(5 * time.Second)
	select {
	case <-timeout:
		s.Fail("Timed out waiting for subscription data")
	default:
		_, recMsg, err = c.ReadMessage()
		s.NoError(err)

		var data map[string]interface{}
		err = json.Unmarshal(recMsg, &data)
		s.NoError(err)

		s.Equal("next", data["type"])
		s.Equal(subscriptionQuery["id"], data["id"])
		s.NotZero(data["payload"].(map[string]interface{})["data"].(map[string]interface{})["time"])
	}

	stopMsg := map[string]interface{}{
		"type": "complete",
		"id":   subscriptionQuery["id"],
	}
	_ = c.WriteJSON(stopMsg)
	_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "test done"))
}

func (s *GraphSuite) Test_Product() {
	var productID string
	var productID2 string

	tests := []struct {
		name  string
		query string
		vars  map[string]any
		check func(body io.Reader)
	}{
		{
			name:  "Add Product",
			query: `mutation { addProduct { id } }`,
			check: func(body io.Reader) {
				result := s.getMap(body)["data"].(map[string]any)["addProduct"].(map[string]any)
				s.NotEmpty(result["id"], "Product ID should not be empty")
				productID = result["id"].(string)
			},
		},
		{
			name:  "Update Product Name",
			query: `mutation($id: String!, $name: String) { updateProduct(id: $id, name: $name) { msg } }`,
			vars:  map[string]any{"id": &productID, "name": "Updated Product"},
			check: func(body io.Reader) {
				result := s.getMap(body)["data"].(map[string]any)["updateProduct"].(map[string]any)
				s.Equal("Product updated", result["msg"], "Update should be successful")
			},
		},
		{
			name:  "Update Product Price",
			query: `mutation($id: String!, $price: Float) { updateProduct(id: $id, price: $price) { msg } }`,
			vars:  map[string]any{"id": &productID, "price": 99.99},
			check: func(body io.Reader) {
				result := s.getMap(body)["data"].(map[string]any)["updateProduct"].(map[string]any)
				s.Equal("Product updated", result["msg"], "Update should be successful")
			},
		},
		{
			name:  "Update Product Nothing to Update",
			query: `mutation($id: String!) { updateProduct(id: $id) { msg } }`,
			vars:  map[string]any{"id": &productID},
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Len(result["errors"], 1)
				s.Equal(model.ErrorNoUpdateParams.Error(), result["errors"].([]any)[0].(map[string]any)["message"],
					"Update should fail")
			},
		},
		{
			name:  "Update Product Not Exist",
			query: `mutation($id: String!, $price: Float) { updateProduct(id: $id, price: $price) { msg } }`,
			vars:  map[string]any{"id": "123", "price": 99.99},
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Len(result["errors"], 1)
				s.Equal(model.ErrorNoRowsUpdated.Error(), result["errors"].([]any)[0].(map[string]any)["message"],
					"Update should fail")
			},
		},
		{
			name:  "Get Product",
			query: `query($id: String!) { getProduct(filter: {id: $id}) { id name price createdAt } }`,
			vars:  map[string]any{"id": &productID},
			check: func(body io.Reader) {
				result := s.getMap(body)["data"].(map[string]any)["getProduct"].(map[string]any)
				s.Equal(productID, result["id"], "Retrieved product ID should match")
				s.Equal("Updated Product", result["name"], "Retrieved product name should match")
				s.Equal(99.99, result["price"], "Retrieved product price should match")
				s.NotEmpty(result["createdAt"], "Retrieved product create time should be filled")
			},
		},
		{
			name:  "Get All Products",
			query: `query { getProducts { id name price createdAt } }`,
			check: func(body io.Reader) {
				products := s.getMap(body)["data"].(map[string]any)["getProducts"].([]any)
				s.Len(products, 1)
				s.Equal(productID, products[0].(map[string]any)["id"])
			},
		},
		{
			name:  "Add Product Fill More",
			query: `mutation { addProduct { id } }`,
			check: func(body io.Reader) {
				result := s.getMap(body)["data"].(map[string]any)["addProduct"].(map[string]any)
				s.NotEmpty(result["id"])
				productID2 = result["id"].(string)
			},
		},
		{
			name:  "Get All Products Check 2",
			query: `query { getProducts { id name price createdAt } }`,
			check: func(body io.Reader) {
				products := s.getMap(body)["data"].(map[string]any)["getProducts"].([]any)
				s.Len(products, 2)
			},
		},
		{
			name:  "Delete Product",
			query: `mutation($id: String!) { deleteProduct(id: $id) { msg } }`,
			vars:  map[string]any{"id": &productID},
			check: func(body io.Reader) {
				result := s.getMap(body)["data"].(map[string]any)["deleteProduct"].(map[string]any)
				s.Equal("Product deleted", result["msg"])
			},
		},
		{
			name:  "Delete Product Second",
			query: `mutation($id: String!) { deleteProduct(id: $id) { msg } }`,
			vars:  map[string]any{"id": &productID2},
			check: func(body io.Reader) {
				result := s.getMap(body)["data"].(map[string]any)["deleteProduct"].(map[string]any)
				s.Equal("Product deleted", result["msg"])
			},
		},
		{
			name:  "Get Product Not Exist",
			query: `query($id: String!) { getProduct(filter: {id: $id}) { id } }`,
			vars:  map[string]any{"id": &productID},
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Len(result["errors"], 1)
				s.Equal(sql.ErrNoRows.Error(), result["errors"].([]any)[0].(map[string]any)["message"],
					"Update should fail")
			},
		},
		{
			name:  "Delete Product Not Exist",
			query: `mutation($id: String!) { deleteProduct(id: $id) { msg } }`,
			vars:  map[string]any{"id": &productID},
			check: func(body io.Reader) {
				result := s.getMap(body)
				s.Len(result["errors"], 1)
				s.Equal("no rows deleted", result["errors"].([]any)[0].(map[string]any)["message"],
					"Update should fail")
			},
		},
		{
			name:  "Get All Products Empty",
			query: `query { getProducts { id name price createdAt } }`,
			check: func(body io.Reader) {
				products := s.getMap(body)["data"].(map[string]any)["getProducts"].([]any)
				s.Empty(products)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			bodyBytes, err := json.Marshal(map[string]any{
				"query":     tc.query,
				"variables": tc.vars,
			})
			s.NoError(err)

			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8000/query", bytes.NewBuffer(bodyBytes))
			s.NoError(err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := s.client.Do(req)
			s.NoError(err)
			s.Equal(http.StatusOK, resp.StatusCode)
			defer resp.Body.Close()

			if tc.check != nil {
				tc.check(resp.Body)
			}
		})
	}
}

func (s *GraphSuite) getMap(body io.Reader) map[string]any {
	result := make(map[string]any)
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		s.FailNow(err.Error())
	}
	return result
}

func TestGraphSqlite(t *testing.T) {
	suite.Run(t, &GraphSuite{
		db: "sqlite",
	})
}

func TestGraphMemory(t *testing.T) {
	suite.Run(t, &GraphSuite{
		db: "in_memory",
	})
}

func TestGraphMemory2(t *testing.T) {
	suite.Run(t, &GraphSuite{
		db: "in_memory2",
	})
}
