package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/flavioribeiro/donut/internal/entity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHTTPServer(
	c *entity.Config,
	mux *http.ServeMux,
	log *zap.Logger,
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
			log.Sugar().Infow(fmt.Sprintf("Starting HTTP server. Open http://%s to access the demo", srv.Addr),
				"addr", srv.Addr,
			)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func ErrorToHTTP(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func SetCORS(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		allowedHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization,X-CSRF-Token"
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
	}
}
