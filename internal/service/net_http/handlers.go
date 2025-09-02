package net_http

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

func (s *Service) Main(w http.ResponseWriter, r *http.Request) {
	var body []byte
	var err error
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			s.sendError(w, http.StatusInternalServerError, err)
			return
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

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response.Bytes())
}

func (s *Service) AddProduct(w http.ResponseWriter, r *http.Request) {
	id, err := s.db.Add(r.Context())
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err)
		return
	}
	s.sendJson(w, &model.Product{ID: id})
}

func (s *Service) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	prod, err := model.ParseProduct(r.FormValue("id"), r.FormValue("name"), r.FormValue("price"))
	if err != nil {
		s.sendError(w, http.StatusBadRequest, err)
		return
	}
	err = s.db.Update(r.Context(), prod)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, model.ErrorNoRowsUpdated) {
			status = http.StatusBadRequest
		}
		s.sendError(w, status, err)
		return
	}
	s.sendMessage(w, "Product updated")
}

func (s *Service) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if err := s.db.Delete(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, model.ErrorNoRowsDeleted) {
			status = http.StatusBadRequest
		}
		s.sendError(w, status, err)
		return
	}
	s.sendMessage(w, "Product deleted")
}

func (s *Service) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	val, err := s.db.Get(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusBadRequest
		}
		s.sendError(w, status, err)
		return
	}
	s.sendJson(w, val)
}

func (s *Service) GetProducts(w http.ResponseWriter, r *http.Request) {
	val, err := s.db.GetAll(r.Context())
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err)
		return
	}
	s.sendJson(w, val)
}

func (s *Service) sendError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	w.Header().Set(ct, ctJSON)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error":%s}`, strconv.Quote(err.Error()))))
}

func (s *Service) sendMessage(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set(ct, ctJSON)
	_, _ = w.Write([]byte(`{"msg":` + strconv.Quote(msg) + `}`))
}

func (s *Service) sendJson(w http.ResponseWriter, val any) {
	b, err := json.Marshal(val)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set(ct, ctJSON)
	_, _ = w.Write(b)
}

const ct = "Content-Type"
const ctJSON = "application/json; charset=utf-8"
