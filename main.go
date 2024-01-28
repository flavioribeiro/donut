//go:build !js
// +build !js

package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/web"
	"github.com/flavioribeiro/donut/internal/web/handlers"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	enableICEMux := false
	flag.BoolVar(&enableICEMux, "enable-ice-mux", false, "Enable ICE Mux on :8081")
	flag.Parse()

	var c entities.Config
	err := envconfig.Process("donut", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	c.EnableICEMux = enableICEMux

	fx.New(
		// HTTP Server
		fx.Provide(web.NewHTTPServer),

		// HTTP router
		fx.Provide(web.NewServeMux),

		// HTTP handlers
		fx.Provide(handlers.NewSignalingHandler),
		fx.Provide(handlers.NewIndexHandler),

		// ICE mux servers
		fx.Provide(controllers.NewTCPICEServer),
		fx.Provide(controllers.NewUDPICEServer),

		// Controllers
		fx.Provide(controllers.NewWebRTCController),
		fx.Provide(controllers.NewSRTController),
		fx.Provide(controllers.NewStreamingController),

		// Logging, Config
		fx.Provide(zap.NewProduction),
		fx.Provide(func() *entities.Config {
			return &c
		}),

		// Forcing the lifecycle initiation with NewHTTPServer
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
