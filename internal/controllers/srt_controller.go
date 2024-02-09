package controllers

import (
	"context"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/flavioribeiro/donut/internal/entities"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SRTController struct {
	c *entities.Config
	l *zap.SugaredLogger
}

func NewSRTController(c *entities.Config, l *zap.SugaredLogger, lc fx.Lifecycle) (*SRTController, error) {
	// Handle logs
	astisrt.SetLogLevel(astisrt.LogLevel(astisrt.LogLevelNotice))
	astisrt.SetLogHandler(func(ll astisrt.LogLevel, file, area, msg string, line int) {
		l.Infow("SRT",
			"ll", ll,
			"msg", msg,
		)
	})

	// Startup srt
	if err := astisrt.Startup(); err != nil {
		l.Errorw("failed to start up srt",
			"error", err,
		)
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// Clean up
			if err := astisrt.CleanUp(); err != nil {
				l.Errorw("failed to clean up srt",
					"error", err,
				)
				return err
			}
			return nil
		},
	})

	return &SRTController{
		c: c,
		l: l,
	}, nil
}

func (c *SRTController) Connect(cancel context.CancelFunc, params *entities.RequestParams) (*astisrt.Connection, error) {
	c.l.Info("trying to connect srt")

	if err := params.Valid(); err != nil {
		return nil, err
	}

	c.l.Infow("Connecting to SRT ",
		"offer", params.String(),
	)

	conn, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(c.c.SRTConnectionLatencyMS),
			astisrt.WithStreamid(params.SRTStreamID),
			astisrt.WithCongestion("live"),
			astisrt.WithTranstype(astisrt.Transtype(astisrt.TranstypeLive)),
		},

		OnDisconnect: func(conn *astisrt.Connection, err error) {
			c.l.Infow("Canceling SRT",
				"error", err,
			)
			cancel()
		},

		Host: params.SRTHost,
		Port: params.SRTPort,
	})
	if err != nil {
		c.l.Errorw("failed to connect srt",
			"error", err,
		)
		return nil, err
	}
	c.l.Infow("Connected to SRT")
	return conn, nil
}
