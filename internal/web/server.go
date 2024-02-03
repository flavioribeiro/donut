package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/flavioribeiro/donut/internal/entities"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHTTPServer(
	c *entities.Config,
	mux *http.ServeMux,
	log *zap.SugaredLogger,
	lc fx.Lifecycle,
) *http.Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort),
		Handler: mux,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Infow(fmt.Sprintf("Starting HTTP server. Open http://%s to access the demo", srv.Addr),
				"addr", srv.Addr,
			)
			// profiling server
			go func() {
				http.ListenAndServe(fmt.Sprintf(":%d", c.PproffHTTPPort), nil)
			}()

			// main server
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
