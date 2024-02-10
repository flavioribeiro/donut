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

var controller *probers.SrtMpegTs

func setupController(t *testing.T) *probers.SrtMpegTs {
	if controller != nil {
		return controller
	}
	fxtest.New(t,
		web.Dependencies(false),
		fx.Populate(&controller),
	)
	return controller
}

func TestSrtMpegTs_StreamInfo(t *testing.T) {
	ffmpeg := teststreaming.FFMPEG_LIVE_SRT_MPEG_TS_H264_AAC

	defer ffmpeg.Stop()
	ffmpeg.Start()

	controller = setupController(t)

	streams, err := controller.StreamInfo(&entities.RequestParams{
		SRTHost:     ffmpeg.Output().Host,
		SRTPort:     uint16(ffmpeg.Output().Port),
		SRTStreamID: "test_id",
	})

	assert.Nil(t, err)
	assert.NotNil(t, streams)
	assert.Equal(t, ffmpeg.ExpectedStreams()[entities.H264], streams[entities.H264])
	assert.Equal(t, ffmpeg.ExpectedStreams()[entities.AAC], streams[entities.AAC])
}

func TestSrtMpegTs_StreamInfo_265(t *testing.T) {
	ffmpeg := teststreaming.FFMPEG_LIVE_SRT_MPEG_TS_H265_AAC

	defer ffmpeg.Stop()
	ffmpeg.Start()

	controller = setupController(t)

	streams, err := controller.StreamInfo(&entities.RequestParams{
		SRTHost:     ffmpeg.Output().Host,
		SRTPort:     uint16(ffmpeg.Output().Port),
		SRTStreamID: "test_id",
	})

	assert.Nil(t, err)
	assert.NotNil(t, streams)
	assert.Equal(t, ffmpeg.ExpectedStreams()[entities.H265], streams[entities.H265])
	assert.Equal(t, ffmpeg.ExpectedStreams()[entities.AAC], streams[entities.AAC])
}
