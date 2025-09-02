package gin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
	gin.SetMode(gin.ReleaseMode)
	mux := gin.New()
	ret := &Service{
		db:  db,
		lis: lis,
	}
	ret.server = &http.Server{
		Handler: mux.Handler(),
	}
	mux.Any("/", ret.Main)
	mux.POST("/add", ret.AddProduct)
	mux.PUT("/update", ret.UpdateProduct)
	mux.DELETE("/delete", ret.DeleteProduct)
	mux.GET("/get", ret.GetProduct)
	mux.GET("/get_all", ret.GetProducts)
	mux.GET("/swagger.yaml", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/yaml", api.SwaggerConfig)
	})

	mux.StaticFS("/swagger/", http.FS(swagger.Swagger))
	return ret, nil
}

func (s *Service) Start(ctx context.Context) error {
	log.Info().Msg("starting gin server")
	go func() {
		if err := s.server.Serve(s.lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start gin server")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("gracefully shutting down gin server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return s.server.Shutdown(ctx)
}
