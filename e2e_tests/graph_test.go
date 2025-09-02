package e2e_tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type GraphSuite struct {
	suite.Suite
}

func (s *GraphSuite) Test_QueryPlayground() {
	res, err := http.Get("http://app-test:8000/query_playground")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	res.Body.Close()
}

func (s *GraphSuite) Test_SubscriptionPlayground() {
	res, err := http.Get("http://app-test:8000/subscription_playground")
	s.NoError(err)
	s.Equal(200, res.StatusCode)
	res.Body.Close()
}

func (s *GraphSuite) Test_Main() {
	res, err := http.Post("http://app-test:8000/query", "application/json",
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
	u, err := url.Parse("ws://app-test:8000/subscription")
	s.NoError(err)

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Subprotocols:     []string{"graphql-transport-ws"},
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

			req, err := http.NewRequest(http.MethodPost, "http://app-test:8000/query", bytes.NewBuffer(bodyBytes))
			s.NoError(err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
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
