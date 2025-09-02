package net_http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/aleksandrzhukovskii/go-template/api"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
	"github.com/aleksandrzhukovskii/go-template/web/swagger"
)

type Service struct {
	server *http.Server
	db     model.DB
	lis    net.Listener
}

func New(db model.DB, lis net.Listener) (model.Server, error) {
	mux := http.NewServeMux()
	ret := &Service{
		db:  db,
		lis: lis,
	}
	ret.server = &http.Server{
		Handler: mux,
	}
	mux.HandleFunc("/", ret.Main)
	mux.HandleFunc("/add", ret.AddProduct)
	mux.HandleFunc("/update", ret.UpdateProduct)
	mux.HandleFunc("/delete", ret.DeleteProduct)
	mux.HandleFunc("/get", ret.GetProduct)
	mux.HandleFunc("/get_all", ret.GetProducts)
	mux.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write(api.SwaggerConfig)
	})
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swagger.Swagger))))
	return ret, nil
}

func (s *Service) Start(ctx context.Context) error {
	log.Info().Msg("starting net/http server")
	go func() {
		if err := s.server.Serve(s.lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start net/http server")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("gracefully shutting down net/http server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return s.server.Shutdown(ctx)
}
