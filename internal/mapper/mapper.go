package mapper

import (
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/pion/webrtc/v3"
)

func FromTrackToRTPCodecCapability(track entities.Track) webrtc.RTPCodecCapability {
	response := webrtc.RTPCodecCapability{}

	if track.Type == entities.H264 {
		response.MimeType = webrtc.MimeTypeH264
	}

	return response
}
