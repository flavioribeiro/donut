package mapper

import (
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/pion/webrtc/v3"
)

func FromTrackToRTPCodecCapability(track entities.Stream) webrtc.RTPCodecCapability {
	response := webrtc.RTPCodecCapability{}

	if track.Codec == entities.H264 {
		response.MimeType = webrtc.MimeTypeH264
	}

	return response
}

func FromMpegTsStreamTypeToCodec(st astits.StreamType) entities.Codec {
	if st == astits.StreamTypeH264Video {
		return entities.H264
	}
	if st == astits.StreamTypeAACAudio {
		return entities.AAC
	}
	return entities.UnknownCodec
}

func FromMpegTsStreamTypeToType(st astits.StreamType) entities.MediaType {
	if st.IsVideo() {
		return entities.VideoType
	}
	if st.IsAudio() {
		return entities.AudioType
	}
	return entities.UnknownType
}
