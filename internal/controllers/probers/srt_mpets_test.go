package probers_test

import (
	"testing"

	"github.com/flavioribeiro/donut/internal/controllers/probers"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/teststreaming"
	"github.com/flavioribeiro/donut/internal/web"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

var p []probers.DonutProber

func setupController(t *testing.T, req *entities.RequestParams) probers.DonutProber {
	if p == nil {
		fxtest.New(t,
			web.Dependencies(false),
			fx.Populate(
				fx.Annotate(
					&p,
					fx.ParamTags(`group:"probers"`),
				),
			),
		)
	}
	for _, c := range p {
		if c.Match(req) {
			return c
		}
	}
	return nil
}

func TestSrtMpegTs_StreamInfo(t *testing.T) {
	t.Parallel()
	ffmpeg := teststreaming.FFMPEG_LIVE_SRT_MPEG_TS_H264_AAC

	defer ffmpeg.Stop()
	ffmpeg.Start()

	req := &entities.RequestParams{
		SRTHost:     ffmpeg.Output().Host,
		SRTPort:     uint16(ffmpeg.Output().Port),
		SRTStreamID: "test_id",
	}

	controller := setupController(t, req)

	streamInfo, err := controller.StreamInfo(req)

	assert.Nil(t, err)
	assert.NotNil(t, streamInfo)
	assert.ElementsMatch(t, ffmpeg.ExpectedStreams(), streamInfo.Streams)
}

func TestSrtMpegTs_StreamInfo_265(t *testing.T) {
	t.Parallel()
	ffmpeg := teststreaming.FFMPEG_LIVE_SRT_MPEG_TS_H265_AAC

	defer ffmpeg.Stop()
	ffmpeg.Start()

	req := &entities.RequestParams{
		SRTHost:     ffmpeg.Output().Host,
		SRTPort:     uint16(ffmpeg.Output().Port),
		SRTStreamID: "test_id",
	}

	controller := setupController(t, req)

	streamInfo, err := controller.StreamInfo(req)

	assert.Nil(t, err)
	assert.NotNil(t, streamInfo)
	assert.ElementsMatch(t, ffmpeg.ExpectedStreams(), streamInfo.Streams)
}
