//go:build !js
// +build !js

package main

import (
	"flag"
	"net/http"

	"github.com/flavioribeiro/donut/internal/web"

	"go.uber.org/fx"
)

func main() {
	enableICEMux := false
	flag.BoolVar(&enableICEMux, "enable-ice-mux", false, "Enable ICE Mux on :8081")
	flag.Parse()

	fx.New(
		web.Dependencies(enableICEMux),
		// Forcing the lifecycle initiation with NewHTTPServer
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
