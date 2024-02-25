package mapper

import (
	"strings"

	"github.com/asticode/go-astiav"
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

// TODO: split mapper by subject (either by files alone or new modules)
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
		m.l.Info("[[[[TODO: mapper not implemented]]]] for ", track)
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
	m.l.Info("[[[[TODO: mapper not implemented]]]] for ", st)
	return entities.UnknownCodec
}

func (m *Mapper) FromMpegTsStreamTypeToType(st astits.StreamType) entities.MediaType {
	if st.IsVideo() {
		return entities.VideoType
	}
	if st.IsAudio() {
		return entities.AudioType
	}
	m.l.Info("[[[[TODO: mapper not implemented]]]] for ", st)
	return entities.UnknownType
}

func (m *Mapper) FromStreamTypeToEntityStream(es *astits.PMTElementaryStream) entities.Stream {
	return entities.Stream{
		Codec: m.FromMpegTsStreamTypeToCodec(es.StreamType),
		Type:  m.FromMpegTsStreamTypeToType(es.StreamType),
		Id:    m.FromMpegTsStreamTypeToID(es),
	}
}

func (m *Mapper) FromMpegTsStreamTypeToID(es *astits.PMTElementaryStream) uint16 {
	return es.ElementaryPID
}

func (m *Mapper) FromWebRTCSessionDescriptionToStreamInfo(desc webrtc.SessionDescription) (*entities.StreamInfo, error) {
	sdpDesc, err := desc.Unmarshal()
	if err != nil {
		return nil, err
	}
	result := &entities.StreamInfo{}
	unique := map[entities.Codec]entities.Stream{}

	for _, desc := range sdpDesc.MediaDescriptions {
		// Currently defined media (MediaName.Media) are "audio","video", "text", "application", and "message"
		// ref https://datatracker.ietf.org/doc/html/rfc4566#section-5.14
		// ref https://aomediacodec.github.io/av1-rtp-spec/#73-examples
		// ref https://webrtchacks.com/sdp-anatomy/
		if desc.MediaName.Media != "video" && desc.MediaName.Media != "audio" {
			m.l.Info("[[[[TODO: mapper not implemented]]]] for ", desc.MediaName.Media)
			continue
		}

		var mediaType entities.MediaType
		if desc.MediaName.Media == "video" {
			mediaType = entities.VideoType
		}
		if desc.MediaName.Media == "audio" {
			mediaType = entities.AudioType
		}

		for _, a := range desc.Attributes {
			if strings.Contains(a.Key, "rtpmap") {
				// Samples:
				// Key:rtpmap Value: 98  VP9/90000
				// Key:rtpmap Value: 102 H264/90000
				// Key:rtpmap Value: 102 H264/90000
				// Key:rtpmap Value: 47  AV1/90000
				// Key:rtpmap Value: 111 opus/48000/2
				if strings.Contains(a.Value, "H264") {
					unique[entities.H264] = entities.Stream{
						Codec: entities.H264,
						Type:  mediaType,
					}
				} else if strings.Contains(a.Value, "H265") {
					unique[entities.H265] = entities.Stream{
						Codec: entities.H265,
						Type:  mediaType,
					}
				} else if strings.Contains(a.Value, "VP8") {
					unique[entities.VP8] = entities.Stream{
						Codec: entities.VP8,
						Type:  mediaType,
					}
				} else if strings.Contains(a.Value, "VP9") {
					unique[entities.VP9] = entities.Stream{
						Codec: entities.VP9,
						Type:  mediaType,
					}
				} else if strings.Contains(a.Value, "AV1") {
					unique[entities.AV1] = entities.Stream{
						Codec: entities.AV1,
						Type:  mediaType,
					}
				} else if strings.Contains(a.Value, "opus") {
					unique[entities.Opus] = entities.Stream{
						Codec: entities.Opus,
						Type:  mediaType,
					}
				} else {
					m.l.Info("[[[[TODO: mapper not implemented]]]] for ", a.Value)
				}
			}
		}

		for _, v := range unique {
			result.Streams = append(result.Streams, v)
		}
	}
	return result, nil
}

func (m *Mapper) FromStreamInfoToEntityMessages(si *entities.StreamInfo) []entities.Message {
	var result []entities.Message

	for _, s := range si.Streams {
		result = append(result, m.FromStreamToEntityMessage(s))
	}

	return result
}

func (m *Mapper) FromStreamToEntityMessage(st entities.Stream) entities.Message {
	return entities.Message{
		Type:    entities.MessageTypeMetadata,
		Message: string(st.Codec),
	}
}

func (m *Mapper) FromLibAVStreamToEntityStream(libavStream *astiav.Stream) entities.Stream {
	st := entities.Stream{}

	if libavStream.CodecParameters().MediaType() == astiav.MediaTypeAudio {
		st.Type = entities.AudioType
	} else if libavStream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
		st.Type = entities.VideoType
	} else {
		m.l.Info("[[[[TODO: mapper not implemented]]]] for ", libavStream.CodecParameters().MediaType())
		st.Type = entities.UnknownType
	}

	// https://github.com/FFmpeg/FFmpeg/blob/master/libavcodec/codec_desc.c#L34
	if libavStream.CodecParameters().CodecID().Name() == "h264" {
		st.Codec = entities.H264
	} else if libavStream.CodecParameters().CodecID().Name() == "h265" {
		st.Codec = entities.H265
	} else if libavStream.CodecParameters().CodecID().Name() == "hevc" {
		st.Codec = entities.H265
	} else if libavStream.CodecParameters().CodecID().Name() == "av1" {
		st.Codec = entities.AV1
	} else if libavStream.CodecParameters().CodecID().Name() == "aac" {
		st.Codec = entities.AAC
	} else if libavStream.CodecParameters().CodecID().Name() == "vp8" {
		st.Codec = entities.VP8
	} else if libavStream.CodecParameters().CodecID().Name() == "vp9" {
		st.Codec = entities.VP9
	} else if libavStream.CodecParameters().CodecID().Name() == "opus" {
		st.Codec = entities.Opus
	} else {
		m.l.Info("[[[[TODO: mapper not implemented]]]] for ", libavStream.CodecParameters().CodecID().Name())
		st.Codec = entities.UnknownCodec
	}

	st.Id = uint16(libavStream.ID())
	st.Index = uint16(libavStream.Index())

	return st
}
