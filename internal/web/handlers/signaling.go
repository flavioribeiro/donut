package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/controllers/probers"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/zap"
)

type SignalingHandler struct {
	c                   *entities.Config
	l                   *zap.SugaredLogger
	webRTCController    *controllers.WebRTCController
	srtController       *controllers.SRTController
	streamingController *controllers.StreamingController
	srtMpegTSprober     *probers.SrtMpegTs
	mapper              *mapper.Mapper
}

func NewSignalingHandler(
	c *entities.Config,
	log *zap.SugaredLogger,
	webRTCController *controllers.WebRTCController,
	srtController *controllers.SRTController,
	streamingController *controllers.StreamingController,
	srtMpegTSprober *probers.SrtMpegTs,
	mapper *mapper.Mapper,
) *SignalingHandler {
	return &SignalingHandler{
		c:                   c,
		l:                   log,
		webRTCController:    webRTCController,
		srtController:       srtController,
		streamingController: streamingController,
		srtMpegTSprober:     srtMpegTSprober,
		mapper:              mapper,
	}
}

func (h *SignalingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return entities.ErrHTTPPostOnly
	}

	params := entities.RequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return err
	}
	if err := params.Valid(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	peer, err := h.webRTCController.CreatePeerConnection(cancel)
	if err != nil {
		return err
	}

	// real stream info from server
	serverStreamInfo, err := h.srtMpegTSprober.StreamInfo(&params)
	if err != nil {
		return err
	}
	// client stream info support from the client (browser)
	// clientStreamInfo, err := h.mapper.FromWebRTCSessionDescriptionToStreamInfo(params.Offer)
	// if err != nil {
	// 	h.l.Errorw("error while fetching server stream info",
	// 		"error", err,
	// 	)
	// 	return err
	// }
	// TODO: create tracks according with SRT available streams
	// for st := range serverStreamInfo.Streams {
	// }

	// Create a video track
	videoTrack, err := h.webRTCController.CreateTrack(
		peer,
		entities.Stream{
			Codec: entities.H264,
		}, "video", params.SRTStreamID,
	)
	if err != nil {
		return err
	}

	// Create a audio track
	// audioTrack, err := h.webRTCController.CreateTrack(
	// 	peer,
	// 	entities.Stream{
	// 		Codec: entities.AAC,
	// 	}, "audio", params.SRTStreamID,
	// )
	// if err != nil {
	// 	h.l.Errorw("error while creating a web rtc track",
	// 		"error", err,
	// 	)
	// 	return err
	// }

	metadataSender, err := h.webRTCController.CreateDataChannel(peer, entities.MetadataChannelID)
	if err != nil {
		return err
	}

	if err = h.webRTCController.SetRemoteDescription(peer, params.Offer); err != nil {
		return err
	}

	localDescription, err := h.webRTCController.GatheringWebRTC(peer)
	if err != nil {
		return err
	}

	srtConnection, err := h.srtController.Connect(cancel, &params)
	if err != nil {
		return err
	}

	go h.streamingController.Stream(&entities.StreamParameters{
		Cancel:        cancel,
		Ctx:           ctx,
		WebRTCConn:    peer,
		SRTConnection: srtConnection,
		VideoTrack:    videoTrack,
		MetadataTrack: metadataSender,
		StreamInfo:    serverStreamInfo,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(*localDescription)
	if err != nil {
		return err
	}

	return nil
}
