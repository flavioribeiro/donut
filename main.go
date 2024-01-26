//go:build !js
// +build !js

package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/flavioribeiro/donut/internal/entity"
	handlers "github.com/flavioribeiro/donut/internal/web"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	enableICEMux := false
	flag.BoolVar(&enableICEMux, "enable-ice-mux", false, "Enable ICE Mux on :8081")
	flag.Parse()

	var c entity.Config
	err := envconfig.Process("donut", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	c.EnableICEMux = enableICEMux

	fx.New(
		fx.Provide(func() *entity.Config {
			return &c
		}),
		fx.Provide(handlers.NewHTTPServer),

		fx.Provide(handlers.NewSignalingHandler),
		fx.Provide(handlers.NewIndexHandler),

		fx.Provide(handlers.NewServeMux),

		fx.Provide(handlers.NewTCPICEServer),
		fx.Provide(handlers.NewUDPICEServer),
		fx.Provide(handlers.NewWebRTCSettingsEngine),
		fx.Provide(handlers.NewWebRTCMediaEngine),

		fx.Provide(zap.NewProduction),

		// just to enforce the lifecycle by using NewHTTPServer
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
