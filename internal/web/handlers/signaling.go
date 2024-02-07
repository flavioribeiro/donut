package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/entities"
	"go.uber.org/zap"
)

type SignalingHandler struct {
	c                   *entities.Config
	l                   *zap.SugaredLogger
	webRTCController    *controllers.WebRTCController
	srtController       *controllers.SRTController
	streamingController *controllers.StreamingController
}

func NewSignalingHandler(
	c *entities.Config,
	log *zap.SugaredLogger,
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

func (h *SignalingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		h.l.Errorw("unexpected method")
		return entities.ErrHTTPPostOnly
	}

	params := entities.RequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		h.l.Errorw("error while decoding request params json",
			"error", err,
		)
		return err
	}
	if err := params.Valid(); err != nil {
		h.l.Errorw("invalid params",
			"error", err,
		)
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	peer, err := h.webRTCController.CreatePeerConnection(cancel)
	if err != nil {
		h.l.Errorw("error while setting up web rtc connection",
			"error", err,
		)
		return err
	}

	// TODO: create tracks according with SRT available streams
	// Create a video track
	videoTrack, err := h.webRTCController.CreateTrack(
		peer,
		entities.Track{
			Type: entities.H264,
		}, "video", params.SRTStreamID,
	)
	if err != nil {
		h.l.Errorw("error while creating a web rtc track",
			"error", err,
		)
		return err
	}

	metadataSender, err := h.webRTCController.CreateDataChannel(peer, entities.MetadataChannelID)
	if err != nil {
		h.l.Errorw("error while creating a web rtc data channel",
			"error", err,
		)
		return err
	}

	if err = h.webRTCController.SetRemoteDescription(peer, params.Offer); err != nil {
		h.l.Errorw("error while setting a remote web rtc description",
			"error", err,
		)
		return err
	}

	localDescription, err := h.webRTCController.GatheringWebRTC(peer)
	if err != nil {
		h.l.Errorw("error while preparing a local web rtc description",
			"error", err,
		)
		return err
	}

	srtConnection, err := h.srtController.Connect(cancel, params)
	if err != nil {
		h.l.Errorw("error while connecting to an srt server",
			"error", err,
		)
		return err
	}

	go h.streamingController.Stream(&entities.StreamParameters{
		Cancel:        cancel,
		Ctx:           ctx,
		WebRTCConn:    peer,
		SRTConnection: srtConnection,
		VideoTrack:    videoTrack,
		MetadataTrack: metadataSender,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(*localDescription)
	if err != nil {
		h.l.Errorw("error while encoding a local web rtc description",
			"error", err,
		)
		return err
	}

	return nil
}
