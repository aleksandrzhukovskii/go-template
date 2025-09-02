package grpc

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

type Service struct {
	UnimplementedProductServiceServer
	db     model.DB
	server *grpc.Server
	lis    net.Listener
}

func New(db model.DB, lis net.Listener) (model.Server, error) {
	ret := &Service{
		db:     db,
		server: grpc.NewServer(),
		lis:    lis,
	}
	RegisterProductServiceServer(ret.server, ret)
	return ret, nil
}

func (s Service) Start(ctx context.Context) error {
	log.Info().Msgf("starting grpc server")
	go func() {
		if err := s.server.Serve(s.lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start grpc server")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("gracefully shutting down grpc server")
	s.server.GracefulStop()
	return nil
}
