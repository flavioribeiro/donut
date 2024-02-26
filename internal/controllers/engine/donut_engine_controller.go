package engine

import (
	"fmt"

	"github.com/flavioribeiro/donut/internal/controllers/probers"
	"github.com/flavioribeiro/donut/internal/controllers/streamers"
	"github.com/flavioribeiro/donut/internal/entities"
	"go.uber.org/fx"
)

type DonutEngine interface {
	Prober() probers.DonutProber
	Streamer() streamers.DonutStreamer
	RecipeFor(server, client *entities.StreamInfo) *entities.DonutTransformRecipe
}

type DonutEngineParams struct {
	fx.In
	Streamers []streamers.DonutStreamer `group:"streamers"`
	Probers   []probers.DonutProber     `group:"probers"`
}

type DonutEngineController struct {
	p DonutEngineParams
}

func NewDonutEngineController(p DonutEngineParams) *DonutEngineController {
	return &DonutEngineController{p}
}

func (c *DonutEngineController) EngineFor(req *entities.RequestParams) (DonutEngine, error) {
	prober := c.selectProberFor(req)
	if prober == nil {
		return nil, fmt.Errorf("request %v: not fulfilled error %w", req, entities.ErrMissingProber)
	}

	streamer := c.selectStreamerFor(req)
	if prober == nil {
		return nil, fmt.Errorf("request %v: not fulfilled error %w", req, entities.ErrMissingStreamer)
	}

	return &donutEngine{
		prober:   prober,
		streamer: streamer,
	}, nil
}

// TODO: try to use generics
func (c *DonutEngineController) selectProberFor(req *entities.RequestParams) probers.DonutProber {
	for _, p := range c.p.Probers {
		if p.Match(req) {
			return p
		}
	}
	return nil
}

// TODO: try to use generics
func (c *DonutEngineController) selectStreamerFor(req *entities.RequestParams) streamers.DonutStreamer {
	for _, p := range c.p.Streamers {
		if p.Match(req) {
			return p
		}
	}
	return nil
}

type donutEngine struct {
	prober   probers.DonutProber
	streamer streamers.DonutStreamer
}

func (d *donutEngine) Prober() probers.DonutProber {
	return d.prober
}

func (d *donutEngine) Streamer() streamers.DonutStreamer {
	return d.streamer
}

func (d *donutEngine) RecipeFor(server, client *entities.StreamInfo) *entities.DonutTransformRecipe {
	// TODO: implement proper matching
	r := &entities.DonutTransformRecipe{
		Video: entities.DonutMediaTask{
			Action: entities.DonutBypass,
		},
		Audio: entities.DonutMediaTask{
			Action: entities.DonutTranscode,
			Codec:  entities.Opus,
			// TODO: create method list options per Codec
			CodecContextOptions: []entities.LibAVOptionsCodecContext{
				// opus specifically works under 48000 Hz
				entities.SetSampleRate(48000),
			},
		},
	}

	return r
}
