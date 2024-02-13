package streammiddlewares

import (
	"encoding/json"

	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/fx"
)

type eia608Middleware struct{}

type EIA608Response struct {
	fx.Out
	EIA608Middleware entities.StreamMiddleware `group:"middlewares"`
}

// NewEIA608 creates a new EIA608 middleware
func NewEIA608() EIA608Response {
	return EIA608Response{
		EIA608Middleware: &eia608Middleware{},
	}
}

// Act parses and send eia608 data from mpeg-ts to metadata channel
func (*eia608Middleware) Act(mpegTSDemuxData *astits.DemuxerData, sp *entities.StreamParameters) error {
	vs := sp.StreamInfo.VideoStreams()
	eia608Reader := newEIA608Reader()

	for _, v := range vs {
		if mpegTSDemuxData.PES != nil && v.Codec == entities.H264 {
			captions, err := eia608Reader.parse(mpegTSDemuxData.PES)
			if err != nil {
				return err
			}

			if captions != "" {
				captionsMsg, err := eia608Reader.buildCaptionsMessage(mpegTSDemuxData.PES.Header.OptionalHeader.PTS, captions)
				if err != nil {
					return err
				}
				sp.MetadataTrack.SendText(captionsMsg)
			}
		}
	}
	return nil
}

type streamInfoMiddleware struct {
	m *mapper.Mapper
}

type StreamInfoResponse struct {
	fx.Out
	StreamInfoMiddleware entities.StreamMiddleware `group:"middlewares"`
}

// NewStreamInfo creates a new StreamInfo middleware
func NewStreamInfo(m *mapper.Mapper) StreamInfoResponse {
	return StreamInfoResponse{
		StreamInfoMiddleware: &streamInfoMiddleware{m: m},
	}
}

// Act parses and send StreamInfo data from mpeg-ts to metadata channel
func (s *streamInfoMiddleware) Act(mpegTSDemuxData *astits.DemuxerData, sp *entities.StreamParameters) error {
	var streams []entities.Stream
	// TODO: check if it makes sense to move this code to a mapper
	if mpegTSDemuxData.PMT != nil {
		for _, es := range mpegTSDemuxData.PMT.ElementaryStreams {
			streams = append(streams, s.m.FromStreamTypeToEntityStream(es))
		}
	}

	msgs := s.m.FromStreamInfoToEntityMessages(&entities.StreamInfo{Streams: streams})
	for _, m := range msgs {
		msg, err := json.Marshal(m)
		if err != nil {
			return err
		}
		sp.MetadataTrack.SendText(string(msg))
	}

	return nil
}
