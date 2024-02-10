package web

import (
	"log"

	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/controllers/probers"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"github.com/flavioribeiro/donut/internal/web/handlers"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Dependencies(enableICEMux bool) fx.Option {
	var c entities.Config
	err := envconfig.Process("donut", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	c.EnableICEMux = enableICEMux

	return fx.Options(
		// HTTP Server
		fx.Provide(NewHTTPServer),

		// HTTP router
		fx.Provide(NewServeMux),

		// HTTP handlers
		fx.Provide(handlers.NewSignalingHandler),
		fx.Provide(handlers.NewIndexHandler),

		// ICE mux servers
		fx.Provide(controllers.NewTCPICEServer),
		fx.Provide(controllers.NewUDPICEServer),

		// Controllers
		fx.Provide(controllers.NewSRTController),
		fx.Provide(controllers.NewStreamingController),

		fx.Provide(controllers.NewWebRTCController),
		fx.Provide(controllers.NewWebRTCSettingsEngine),
		fx.Provide(controllers.NewWebRTCMediaEngine),
		fx.Provide(controllers.NewWebRTCAPI),
		fx.Provide(probers.NewSrtMpegTs),

		// Mappers
		fx.Provide(mapper.NewMapper),

		// Logging, Config constructors
		fx.Provide(func() *zap.SugaredLogger {
			logger, _ := zap.NewProduction()
			return logger.Sugar()
		}),
		fx.Provide(func() *entities.Config {
			return &c
		}),
	)
}