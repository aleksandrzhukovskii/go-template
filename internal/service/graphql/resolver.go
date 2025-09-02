package graphql

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type requestKey string

var (
	urlKey    = requestKey("url")
	ipKey     = requestKey("ip")
	methodKey = requestKey("method")
)

type Resolver struct {
	server *http.Server
	db     model.DB
	lis    net.Listener
}

func New(db model.DB, lis net.Listener) (model.Server, error) {
	ret := &Resolver{
		db:  db,
		lis: lis,
	}
	c := Config{Resolvers: ret}
	es := NewExecutableSchema(c)
	mux := http.NewServeMux()
	mux.Handle("/query_playground", playground.Handler("Query playground", "/query"))
	mux.Handle("/subscription_playground", playground.Handler("Subscription playground", "/subscription"))
	graph := NewServer(es)
	mux.Handle("/query", middleware(graph))
	mux.Handle("/subscription", middleware(graph))

	ret.server = &http.Server{
		Handler: mux,
	}

	return ret, nil
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(
			context.WithValue(
				context.WithValue(r.Context(), ipKey, r.RemoteAddr),
				urlKey, r.URL.String(),
			), methodKey, r.Method,
		)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (r *Resolver) Start(ctx context.Context) error {
	log.Info().Msg("starting graphql server")
	go func() {
		if err := r.server.Serve(r.lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start graphql server")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("gracefully shutting down graphql server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return r.server.Shutdown(ctx)
}

func NewServer(es graphql.ExecutableSchema) *handler.Server {
	srv := handler.New(es)

	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		InitTimeout:           30 * time.Second,
		KeepAlivePingInterval: 15 * time.Second,
	})

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	return srv
}
