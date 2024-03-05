package engine

import (
	"fmt"

	"github.com/flavioribeiro/donut/internal/controllers/probers"
	"github.com/flavioribeiro/donut/internal/controllers/streamers"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/fx"
)

type DonutEngine interface {
	ServerIngredients(req *entities.RequestParams) (*entities.StreamInfo, error)
	ClientIngredients(req *entities.RequestParams) (*entities.StreamInfo, error)
	RecipeFor(req *entities.RequestParams, server, client *entities.StreamInfo) *entities.DonutRecipe
	Serve(p *entities.DonutParameters)
}

type DonutEngineParams struct {
	fx.In
	Streamers []streamers.DonutStreamer `group:"streamers"`
	Probers   []probers.DonutProber     `group:"probers"`
	Mapper    *mapper.Mapper
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
	if streamer == nil {
		return nil, fmt.Errorf("request %v: not fulfilled error %w", req, entities.ErrMissingStreamer)
	}

	return &donutEngine{
		prober:   prober,
		streamer: streamer,
		mapper:   c.p.Mapper,
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
	mapper   *mapper.Mapper
}

func (d *donutEngine) ServerIngredients(req *entities.RequestParams) (*entities.StreamInfo, error) {
	return d.prober.StreamInfo(req)
}

func (d *donutEngine) ClientIngredients(req *entities.RequestParams) (*entities.StreamInfo, error) {
	return d.mapper.FromWebRTCSessionDescriptionToStreamInfo(req.Offer)
}

func (d *donutEngine) Serve(p *entities.DonutParameters) {
	d.streamer.Stream(p)
}

func (d *donutEngine) RecipeFor(req *entities.RequestParams, server, client *entities.StreamInfo) *entities.DonutRecipe {
	// TODO: implement proper matching
	//
	// suggestions:
	//  if client.medias.contains(server.media)
	//     bypass, server.media
	//  else
	//     preferable = [vp8, opus]
	//     if union(preferable, client.medias)
	//         transcode, preferable
	r := &entities.DonutRecipe{
		Input: entities.DonutInput{
			Format: "mpegts", // it'll change based on input, i.e. rmtp flv
			Options: map[entities.DonutInputOptionKey]string{
				entities.DonutSRTStreamID:  req.SRTStreamID,
				entities.DonutSRTTranstype: "live",
				entities.DonutSRTsmoother:  "live",
			},
		},
		Video: entities.DonutMediaTask{
			Action: entities.DonutBypass,
			Codec:  entities.H264,
		},
		Audio: entities.DonutMediaTask{
			Action: entities.DonutTranscode,
			Codec:  entities.Opus,
			// TODO: create method list options per Codec
			CodecContextOptions: []entities.LibAVOptionsCodecContext{
				// opus specifically works under 48000 Hz
				entities.SetSampleRate(48000),
				// once we changed the sample rate we need to update the time base
				entities.SetTimeBase(1, 48000),
				// for some reason it's setting "s16"
				// entities.SetSampleFormat("fltp"),
			},
		},
	}

	return r
}
