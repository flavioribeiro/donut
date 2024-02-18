package probers

import (
	"fmt"

	"github.com/asticode/go-astiav"
	"github.com/asticode/go-astikit"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type LibAVFFmpeg struct {
	c *entities.Config
	l *zap.SugaredLogger
	m *mapper.Mapper
}

type ResultLibAVFFmpeg struct {
	fx.Out
	LibAVFFmpegProber DonutProber `group:"probers"`
}

// NewLibAVFFmpeg creates a new LibAVFFmpeg DonutProber
func NewLibAVFFmpeg(
	c *entities.Config,
	l *zap.SugaredLogger,
	m *mapper.Mapper,
) ResultLibAVFFmpeg {
	return ResultLibAVFFmpeg{
		LibAVFFmpegProber: &LibAVFFmpeg{
			c: c,
			l: l,
			m: m,
		},
	}
}

// Match returns true when the request is for an LibAVFFmpeg prober
func (c *LibAVFFmpeg) Match(req *entities.RequestParams) bool {
	if req.SRTHost != "" {
		return true
	}
	return false
}

// StreamInfo connects to the SRT stream and probe N packets to discovery the media properties.
func (c *LibAVFFmpeg) StreamInfo(req *entities.RequestParams) (*entities.StreamInfo, error) {
	closer := astikit.NewCloser()
	defer closer.Close()

	var inputFormatContext *astiav.FormatContext
	if inputFormatContext = astiav.AllocFormatContext(); inputFormatContext == nil {
		return nil, entities.ErrFFmpegLibAVFormatContextIsNil
	}
	closer.Add(inputFormatContext.Free)

	// TODO: add an UI element for sub-type (format) when input is srt:// (defaulting to mpeg-ts)
	userProvidedInputFormat := "mpegts"
	// We're assuming that SRT is carrying mpegts.
	//
	// ffmpeg -hide_banner -protocols # shows all protocols (SRT/RTMP)
	// ffmpeg -hide_banner -formats # shows all formats (mpegts/webm/mov)
	inputFormat := astiav.FindInputFormat(userProvidedInputFormat)
	if inputFormat == nil {
		return nil, fmt.Errorf("mpegts: %w", entities.ErrFFmpegLibAVNotFound)
	}

	inputURL := fmt.Sprintf("srt://%s:%d/%s", req.SRTHost, req.SRTPort, req.SRTStreamID)
	if err := inputFormatContext.OpenInput(inputURL, inputFormat, nil); err != nil {
		return nil, fmt.Errorf("error while inputFormatContext.OpenInput: %w", err)
	}

	if err := inputFormatContext.FindStreamInfo(nil); err != nil {
		return nil, fmt.Errorf("error while inputFormatContext.FindStreamInfo %w", err)
	}

	streams := []entities.Stream{}
	for _, is := range inputFormatContext.Streams() {
		if is.CodecParameters().MediaType() != astiav.MediaTypeAudio &&
			is.CodecParameters().MediaType() != astiav.MediaTypeVideo {
			continue
		}

		streams = append(streams, c.m.FromLibAVStreamToEntityStream(is))
	}
	si := entities.StreamInfo{Streams: streams}

	return &si, nil
}
