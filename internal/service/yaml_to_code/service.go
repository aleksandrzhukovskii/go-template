package yaml_to_code

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/rs/zerolog/log"

	"github.com/aleksandrzhukovskii/go-template/api"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
	"github.com/aleksandrzhukovskii/go-template/web/swagger"
)

type implement interface {
	StrictServerInterface
	model.Server
}

var _ implement = &Service{}

type ctxKey string

var reqKey = ctxKey("request")

type Service struct {
	server *http.Server
	db     model.DB
	lis    net.Listener
}

func New(db model.DB, lis net.Listener) (model.Server, error) {
	ret := &Service{
		db:  db,
		lis: lis,
	}
	mux := http.NewServeMux()
	ret.server = &http.Server{
		Handler: mux,
	}
	HandlerFromMux(NewStrictHandler(ret, []StrictMiddlewareFunc{
		func(f strictnethttp.StrictHTTPHandlerFunc, operationID string) strictnethttp.StrictHTTPHandlerFunc {
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
				return f(context.WithValue(ctx, reqKey, r), w, r, request)
			}
		},
	}), mux)
	mux.HandleFunc("GET /swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write(api.SwaggerConfig)
	})
	mux.Handle("GET /swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swagger.Swagger))))
	return ret, nil
}

func (s *Service) Start(ctx context.Context) error {
	log.Info().Msg("starting yaml_to_code server")
	go func() {
		if err := s.server.Serve(s.lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start yaml_to_code server")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("gracefully shutting down yaml_to_code server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return s.server.Shutdown(ctx)
}
