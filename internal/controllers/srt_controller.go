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
	l *zap.Logger
}

func NewSRTController(c *entities.Config, l *zap.Logger, lc fx.Lifecycle) (*SRTController, error) {
	// Handle logs
	astisrt.SetLogLevel(astisrt.LogLevel(astisrt.LogLevelError))
	astisrt.SetLogHandler(func(ll astisrt.LogLevel, file, area, msg string, line int) {
		l.Sugar().Infow("SRT",
			"ll", ll,
			"msg", msg,
		)
	})

	// Startup srt
	if err := astisrt.Startup(); err != nil {
		l.Sugar().Errorw("failed to start up srt",
			"error", err,
		)
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// Clean up
			if err := astisrt.CleanUp(); err != nil {
				l.Sugar().Errorw("failed to clean up srt",
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

func (c *SRTController) Connect(params *entities.RequestParams) (*astisrt.Connection, error) {
	c.l.Sugar().Infow("trying to connect srt")
	if params == nil {
		return nil, entities.ErrMissingRemoteOffer
	}

	if err := params.Valid(); err != nil {
		return nil, err
	}

	c.l.Sugar().Infow("Connecting to SRT ",
		"offer", params,
	)

	conn, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(c.c.SRTConnectionLatencyMS),
			astisrt.WithStreamid(params.SRTStreamID),
			astisrt.WithCongestion("live"),
			astisrt.WithTranstype(astisrt.Transtype(astisrt.TranstypeLive)),
		},

		OnDisconnect: func(conn *astisrt.Connection, err error) {
			c.l.Sugar().Fatalw("Disconnected from SRT",
				"error", err,
			)
		},

		Host: params.SRTHost,
		Port: params.SRTPort,
	})
	if err != nil {
		c.l.Sugar().Errorw("failed to connect srt",
			"error", err,
		)
		return nil, err
	}
	c.l.Sugar().Infow("Connected to SRT")
	return conn, nil
}
