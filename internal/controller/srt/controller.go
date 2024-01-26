package srt

import (
	"log"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/flavioribeiro/donut/internal/entity"
)

type SRTController struct {
	c *entity.Config
}

func NewSRTController(c *entity.Config) *SRTController {
	return &SRTController{
		c: c,
	}
}

func (c *SRTController) Connect(offer *entity.ParamsOffer) error {
	if err := offer.Valid(); err != nil {
		return err
	}

	// conn, err := c.srtConnect(offer)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (c *SRTController) srtConnect(offer *entity.ParamsOffer) (*astisrt.Connection, error) {
	srtConnection, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(c.c.SRTConnectionLatencyMS),
			astisrt.WithStreamid(offer.SRTStreamID),
		},
		OnDisconnect: func(c *astisrt.Connection, err error) {
			log.Fatal("Disconnected from SRT")
		},
		Host: offer.SRTHost,
		Port: offer.SRTPort,
	})
	if err != nil {
		return nil, err
	}
	return srtConnection, nil
}
