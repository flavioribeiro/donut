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
func (c *LibAVFFmpeg) StreamInfo(req entities.DonutAppetizer) (*entities.StreamInfo, error) {
	closer := astikit.NewCloser()
	defer closer.Close()

	var inputFormatContext *astiav.FormatContext
	if inputFormatContext = astiav.AllocFormatContext(); inputFormatContext == nil {
		return nil, entities.ErrFFmpegLibAVFormatContextIsNil
	}
	closer.Add(inputFormatContext.Free)

	inputFormat, err := c.defineInputFormat(req.Format.String())
	if err != nil {
		return nil, err
	}
	inputOptions := c.defineInputOptions(req.Options, closer)

	if err := inputFormatContext.OpenInput(req.URL, inputFormat, inputOptions); err != nil {
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

// TODO: merge common behavior (streamer / prober)
func (c *LibAVFFmpeg) defineInputFormat(streamFormat string) (*astiav.InputFormat, error) {
	var inputFormat *astiav.InputFormat
	if streamFormat != "" {
		inputFormat = astiav.FindInputFormat(streamFormat)
		if inputFormat == nil {
			return nil, fmt.Errorf("ffmpeg/libav: could not find %s input format", streamFormat)
		}
	}
	return inputFormat, nil
}

func (c *LibAVFFmpeg) defineInputOptions(opts map[entities.DonutInputOptionKey]string, closer *astikit.Closer) *astiav.Dictionary {
	var dic *astiav.Dictionary
	if len(opts) > 0 {
		dic = &astiav.Dictionary{}
		closer.Add(dic.Free)

		for k, v := range opts {
			dic.Set(k.String(), v, 0)
		}
	}
	return dic
}
