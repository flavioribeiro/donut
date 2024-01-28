//go:build !js
// +build !js

package main

import (
	"flag"
	"log"
	"net/http"

	donutsrt "github.com/flavioribeiro/donut/internal/controller/srt"
	donutstreaming "github.com/flavioribeiro/donut/internal/controller/streaming"
	donutwebrtc "github.com/flavioribeiro/donut/internal/controller/webrtc"
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
		// Server entry point
		fx.Provide(handlers.NewHTTPServer),

		// HTTP handlers
		fx.Provide(handlers.NewSignalingHandler),
		fx.Provide(handlers.NewIndexHandler),

		// HTTP router
		fx.Provide(handlers.NewServeMux),

		// ICE mux servers
		fx.Provide(donutwebrtc.NewTCPICEServer),
		fx.Provide(donutwebrtc.NewUDPICEServer),

		// WebRTC controller
		fx.Provide(donutwebrtc.NewWebRTCController),
		// SRT controller
		fx.Provide(donutsrt.NewSRTController),
		// Streaming controller
		fx.Provide(donutstreaming.NewStreamingController),

		// Logging, Config
		fx.Provide(zap.NewProduction),
		fx.Provide(func() *entity.Config {
			return &c
		}),

		// Forcing the lifecycle initiation with NewHTTPServer
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
