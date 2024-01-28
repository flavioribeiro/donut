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
		h.l.Sugar().Errorw("unexpected method")
		SetError(w, entities.ErrHTTPPostOnly)
		return
	}

	params := entities.RequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		h.l.Sugar().Errorw("error while decoding request params json",
			"error", err,
		)
		SetError(w, err)
		return
	}
	if err := params.Valid(); err != nil {
		h.l.Sugar().Errorw("invalid params",
			"error", err,
		)
		SetError(w, err)
		return
	}

	if err := h.webRTCController.SetupPeerConnection(); err != nil {
		h.l.Sugar().Errorw("error while setting up web rtc connection",
			"error", err,
		)
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
		h.l.Sugar().Errorw("error while creating a web rtc track",
			"error", err,
		)
		SetError(w, err)
		return
	}

	metadataSender, err := h.webRTCController.CreateDataChannel(entities.MetadataChannelID)
	if err != nil {
		h.l.Sugar().Errorw("error while createing a web rtc data channel",
			"error", err,
		)
		SetError(w, err)
	}

	if err = h.webRTCController.SetRemoteDescription(params.Offer); err != nil {
		h.l.Sugar().Errorw("error while setting a remote web rtc description",
			"error", err,
		)
		SetError(w, err)
		return
	}

	localDescription, err := h.webRTCController.GatheringWebRTC()
	if err != nil {
		h.l.Sugar().Errorw("error while preparing a local web rtc description",
			"error", err,
		)
		SetError(w, err)
		return
	}

	localOfferDescription, err := json.Marshal(*localDescription)
	if err != nil {
		h.l.Sugar().Errorw("error while encoding a local web rtc description",
			"error", err,
		)
		SetError(w, err)
		return
	}

	srtConnection, err := h.srtController.Connect(&params)
	if err != nil {
		h.l.Sugar().Errorw("error while connecting to an srt server",
			"error", err,
		)
		SetError(w, err)
		return
	}

	go h.streamingController.Stream(srtConnection, videoTrack, metadataSender)

	if _, err := w.Write(localOfferDescription); err != nil {
		h.l.Sugar().Errorw("error responding the local web rtc offer description",
			"error", err,
		)
		SetError(w, err)
		return
	}
	SetSuccessJson(w)
}
