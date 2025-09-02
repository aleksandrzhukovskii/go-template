package server_tests

import (
	"context"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/test/bufconn"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
	"github.com/aleksandrzhukovskii/go-template/internal/service"
)

func MainAnalogue(ctx context.Context, lis *bufconn.Listener) {
	cfg, err := config.New()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config")
	}

	services, err := service.NewWithListener(cfg, lis)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to prepare services")
	}

	if err = services.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start services or an issue while stopping them")
	}
}
