package mapper

import (
	"github.com/flavioribeiro/donut/internal/entity"
	"github.com/pion/webrtc/v3"
)

func FromTrackToRTPCodecCapability(track entity.Track) webrtc.RTPCodecCapability {
	response := webrtc.RTPCodecCapability{}

	if track.Type == entity.H264 {
		response.MimeType = webrtc.MimeTypeH264
	}

	return response
}
