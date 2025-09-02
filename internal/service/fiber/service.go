package fiber

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/rs/zerolog/log"

	"github.com/aleksandrzhukovskii/go-template/api"
	"github.com/aleksandrzhukovskii/go-template/internal/model"
	"github.com/aleksandrzhukovskii/go-template/web/swagger"
)

type Service struct {
	server *fiber.App
	db     model.DB
	lis    net.Listener
}

func New(db model.DB, lis net.Listener) (model.Server, error) {
	ret := &Service{
		server: fiber.New(fiber.Config{
			DisableStartupMessage: true,
		}),
		db:  db,
		lis: lis,
	}
	ret.server.All("/", ret.Main)
	ret.server.Post("/add", ret.AddProduct)
	ret.server.Put("/update", ret.UpdateProduct)
	ret.server.Delete("/delete", ret.DeleteProduct)
	ret.server.Get("/get", ret.GetProduct)
	ret.server.Get("/get_all", ret.GetProducts)
	ret.server.Get("/swagger.yaml", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).Type("text/yaml").Send(api.SwaggerConfig)
	})
	ret.server.Use("/swagger/", filesystem.New(filesystem.Config{
		Root: http.FS(swagger.Swagger),
	}))
	return ret, nil
}

func (s *Service) Start(ctx context.Context) error {
	log.Info().Msg("starting fiber server")
	go func() {
		if err := s.server.Listener(s.lis); err != nil && !errors.Is(err, http.ErrServerClosed) && !strings.Contains(err.Error(), "closed") {
			log.Fatal().Err(err).Msg("failed to start fiber server")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("gracefully shutting down fiber server")
	return s.server.Shutdown()
}
