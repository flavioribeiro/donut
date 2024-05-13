package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/controllers/engine"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/zap"
)

type SignalingHandler struct {
	c                *entities.Config
	l                *zap.SugaredLogger
	webRTCController *controllers.WebRTCController
	mapper           *mapper.Mapper
	donut            *engine.DonutEngineController
}

func NewSignalingHandler(
	c *entities.Config,
	log *zap.SugaredLogger,
	webRTCController *controllers.WebRTCController,
	mapper *mapper.Mapper,
	donut *engine.DonutEngineController,
) *SignalingHandler {
	return &SignalingHandler{
		c:                c,
		l:                log,
		webRTCController: webRTCController,
		mapper:           mapper,
		donut:            donut,
	}
}

func (h *SignalingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	params, err := h.createAndValidateParams(r)
	if err != nil {
		return err
	}
	h.l.Infof("RequestParams %s", params.String())

	donutEngine, err := h.donut.EngineFor(&params)
	if err != nil {
		return err
	}
	h.l.Infof("DonutEngine %#v", donutEngine)

	// server side media info
	serverStreamInfo, err := donutEngine.ServerIngredients()
	if err != nil {
		return err
	}
	h.l.Infof("ServerIngredients %#v", serverStreamInfo)

	// client side media support
	clientStreamInfo, err := donutEngine.ClientIngredients()
	if err != nil {
		return err
	}
	h.l.Infof("ClientIngredients %#v", clientStreamInfo)

	donutRecipe, err := donutEngine.RecipeFor(serverStreamInfo, clientStreamInfo)
	if err != nil {
		return err
	}
	h.l.Infof("DonutRecipe %#v", donutRecipe)

	// We can't defer calling cancel here because it'll live alongside the stream.
	ctx, cancel := context.WithCancel(context.Background())
	webRTCResponse, err := h.webRTCController.Setup(cancel, donutRecipe, params)
	if err != nil {
		cancel()
		return err
	}
	h.l.Infof("WebRTCResponse %#v", webRTCResponse)

	//TODO: remove the sleeping
	// The simulated RTMP stream (/scripts/ffmpeg_rtmp.sh) goes down every time a client disconnects.
	// The prober is forcing the first restart therefore it waits for 4 seconds.
	time.Sleep(4 * time.Second)

	go donutEngine.Serve(&entities.DonutParameters{
		Cancel: cancel,
		Ctx:    ctx,

		Recipe: *donutRecipe,

		OnClose: func() {
			cancel()
			webRTCResponse.Connection.Close()
		},
		OnError: func(err error) {
			h.l.Errorw("error while streaming", "error", err)
		},
		OnStream: func(st *entities.Stream) error {
			return h.webRTCController.SendMetadata(webRTCResponse.Data, st)
		},
		OnVideoFrame: func(data []byte, c entities.MediaFrameContext) error {
			return h.webRTCController.SendMediaSample(webRTCResponse.Video, data, c)
		},
		OnAudioFrame: func(data []byte, c entities.MediaFrameContext) error {
			return h.webRTCController.SendMediaSample(webRTCResponse.Audio, data, c)
		},
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(*webRTCResponse.LocalSDP)
	if err != nil {
		cancel()
		return err
	}
	h.l.Infof("webRTCResponse %#v", webRTCResponse)

	return nil
}

func (h *SignalingHandler) createAndValidateParams(r *http.Request) (entities.RequestParams, error) {
	if r.Method != http.MethodPost {
		return entities.RequestParams{}, entities.ErrHTTPPostOnly
	}

	params := entities.RequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return entities.RequestParams{}, err
	}
	if err := params.Valid(); err != nil {
		return entities.RequestParams{}, err
	}

	return params, nil
}
