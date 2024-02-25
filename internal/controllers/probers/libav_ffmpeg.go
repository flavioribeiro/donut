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
	return req.SRTHost != ""
}

// StreamInfo connects to the SRT stream to discovery media properties.
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

	// ref https://ffmpeg.org/ffmpeg-all.html#srt
	d := &astiav.Dictionary{}
	// flags (the zeroed 3rd value) https://github.com/FFmpeg/FFmpeg/blob/n5.0/libavutil/dict.h#L67C9-L77
	d.Set("srt_streamid", req.SRTStreamID, 0)
	d.Set("smoother", "live", 0)
	d.Set("transtype", "live", 0)

	inputURL := fmt.Sprintf("srt://%s:%d", req.SRTHost, req.SRTPort)
	if err := inputFormatContext.OpenInput(inputURL, inputFormat, d); err != nil {
		return nil, fmt.Errorf("error while inputFormatContext.OpenInput: %w", err)
	}
	closer.Add(inputFormatContext.CloseInput)

	if err := inputFormatContext.FindStreamInfo(nil); err != nil {
		return nil, fmt.Errorf("error while inputFormatContext.FindStreamInfo %w", err)
	}

	streams := []entities.Stream{}
	for _, is := range inputFormatContext.Streams() {
		if is.CodecParameters().MediaType() != astiav.MediaTypeAudio &&
			is.CodecParameters().MediaType() != astiav.MediaTypeVideo {
			c.l.Info("skipping media type", is.CodecParameters().MediaType())
			continue
		}
		streams = append(streams, c.m.FromLibAVStreamToEntityStream(is))
	}
	si := entities.StreamInfo{Streams: streams}

	return &si, nil
}
