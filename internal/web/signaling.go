package handlers

import (
	"encoding/json"
	"net/http"

	donutsrt "github.com/flavioribeiro/donut/internal/controller/srt"
	donutstreaming "github.com/flavioribeiro/donut/internal/controller/streaming"
	donutwebrtc "github.com/flavioribeiro/donut/internal/controller/webrtc"
	"github.com/flavioribeiro/donut/internal/entity"
	"go.uber.org/zap"
)

type SignalingHandler struct {
	c                   *entity.Config
	l                   *zap.Logger
	webRTCController    *donutwebrtc.WebRTCController
	srtController       *donutsrt.SRTController
	streamingController *donutstreaming.StreamingController
}

func NewSignalingHandler(
	c *entity.Config,
	log *zap.Logger,
	webRTCController *donutwebrtc.WebRTCController,
	srtController *donutsrt.SRTController,
	streamingController *donutstreaming.StreamingController,
) *SignalingHandler {
	return &SignalingHandler{
		c:                   c,
		l:                   log,
		webRTCController:    webRTCController,
		srtController:       srtController,
		streamingController: streamingController,
	}
}

func (h *SignalingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	SetCORS(w, r)
	if r.Method != http.MethodPost {
		ErrorToHTTP(w, entity.ErrHTTPPostOnly)
		return
	}

	params := entity.RequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		ErrorToHTTP(w, err)
		return
	}
	if err := params.Valid(); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	if err := h.webRTCController.SetupPeerConnection(); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	// TODO: create tracks according with SRT available streams
	// Create a video track
	videoTrack, err := h.webRTCController.CreateTrack(
		entity.Track{
			Type: entity.H264,
		}, "video", params.SRTStreamID,
	)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	metadataSender, err := h.webRTCController.CreateDataChannel(entity.MetadataChannelID)
	if err != nil {
		ErrorToHTTP(w, err)
	}

	if err = h.webRTCController.SetRemoteDescription(params.Offer); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	localDescription, err := h.webRTCController.GatheringWebRTC()
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	localOfferDescription, err := json.Marshal(*localDescription)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	srtConnection, err := h.srtController.Connect(&params)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	go h.streamingController.Stream(r.Context(), srtConnection, videoTrack, metadataSender)

	if _, err := w.Write(localOfferDescription); err != nil {
		ErrorToHTTP(w, err)
		return
	}
	SetSuccessJson(w)
}
