package mapper

import (
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type Mapper struct {
	l *zap.SugaredLogger
}

func NewMapper(l *zap.SugaredLogger) *Mapper {
	return &Mapper{l: l}
}

func (m *Mapper) FromTrackToRTPCodecCapability(track entities.Stream) webrtc.RTPCodecCapability {
	response := webrtc.RTPCodecCapability{}

	if track.Codec == entities.H264 {
		response.MimeType = webrtc.MimeTypeH264
	} else if track.Codec == entities.H265 {
		response.MimeType = webrtc.MimeTypeH265
	} else {
		m.l.Info("[[[[TODO: not implemented]]]]", track)
	}

	return response
}

func (m *Mapper) FromMpegTsStreamTypeToCodec(st astits.StreamType) entities.Codec {
	if st == astits.StreamTypeH264Video {
		return entities.H264
	}
	if st == astits.StreamTypeH265Video {
		return entities.H265
	}
	if st == astits.StreamTypeAACAudio {
		return entities.AAC
	}
	m.l.Info("[[[[TODO: not implemented]]]]", st)
	return entities.UnknownCodec
}

func (m *Mapper) FromMpegTsStreamTypeToType(st astits.StreamType) entities.MediaType {
	if st.IsVideo() {
		return entities.VideoType
	}
	if st.IsAudio() {
		return entities.AudioType
	}
	m.l.Info("[[[[TODO: not implemented]]]]", st)
	return entities.UnknownType
}

func (m *Mapper) FromStreamTypeToEntityStream(st astits.StreamType) entities.Stream {
	return entities.Stream{
		Codec: m.FromMpegTsStreamTypeToCodec(st),
		Type:  m.FromMpegTsStreamTypeToType(st),
	}
}
