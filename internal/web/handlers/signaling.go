package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/entities"
	"go.uber.org/zap"
)

type SignalingHandler struct {
	c                   *entities.Config
	l                   *zap.Logger
	webRTCController    *controllers.WebRTCController
	srtController       *controllers.SRTController
	streamingController *controllers.StreamingController
}

func NewSignalingHandler(
	c *entities.Config,
	log *zap.Logger,
	webRTCController *controllers.WebRTCController,
	srtController *controllers.SRTController,
	streamingController *controllers.StreamingController,
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
	if r.Method != http.MethodPost {
		SetError(w, entities.ErrHTTPPostOnly)
		return
	}

	params := entities.RequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		SetError(w, err)
		return
	}
	if err := params.Valid(); err != nil {
		SetError(w, err)
		return
	}

	if err := h.webRTCController.SetupPeerConnection(); err != nil {
		SetError(w, err)
		return
	}

	// TODO: create tracks according with SRT available streams
	// Create a video track
	videoTrack, err := h.webRTCController.CreateTrack(
		entities.Track{
			Type: entities.H264,
		}, "video", params.SRTStreamID,
	)
	if err != nil {
		SetError(w, err)
		return
	}

	metadataSender, err := h.webRTCController.CreateDataChannel(entities.MetadataChannelID)
	if err != nil {
		SetError(w, err)
	}

	if err = h.webRTCController.SetRemoteDescription(params.Offer); err != nil {
		SetError(w, err)
		return
	}

	localDescription, err := h.webRTCController.GatheringWebRTC()
	if err != nil {
		SetError(w, err)
		return
	}

	localOfferDescription, err := json.Marshal(*localDescription)
	if err != nil {
		SetError(w, err)
		return
	}

	srtConnection, err := h.srtController.Connect(&params)
	if err != nil {
		SetError(w, err)
		return
	}

	go h.streamingController.Stream(srtConnection, videoTrack, metadataSender)

	if _, err := w.Write(localOfferDescription); err != nil {
		SetError(w, err)
		return
	}
	SetSuccessJson(w)
}
