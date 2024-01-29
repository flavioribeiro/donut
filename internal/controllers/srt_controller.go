package controllers

import (
	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/flavioribeiro/donut/internal/entities"
	"go.uber.org/zap"
)

type SRTController struct {
	c *entities.Config
	l *zap.Logger
}

func NewSRTController(c *entities.Config, l *zap.Logger) *SRTController {
	return &SRTController{
		c: c,
		l: l,
	}
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
		c.l.Sugar().Infow("failed to connect srt",
			"error", err,
		)
		return nil, err
	}
	c.l.Sugar().Infow("Connected to SRT")
	return conn, nil
}
