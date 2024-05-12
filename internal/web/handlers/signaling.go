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
	h.l.Infof("createAndValidateParams %s", params.String())

	donutEngine, err := h.donut.EngineFor(&params)
	if err != nil {
		return err
	}
	h.l.Infof("EngineFor %#v", donutEngine)

	// server side media info
	serverStreamInfo, err := donutEngine.ServerIngredients()
	if err != nil {
		return err
	}
	// client side media support
	clientStreamInfo, err := donutEngine.ClientIngredients()
	if err != nil {
		return err
	}
	h.l.Infof("ServerIngredients %#v", serverStreamInfo)
	h.l.Infof("ClientIngredients %#v", clientStreamInfo)

	donutRecipe, err := donutEngine.RecipeFor(serverStreamInfo, clientStreamInfo)
	h.l.Info("after RecipeFor")
	h.l.Info("after RecipeFor err", err)
	h.l.Info("after RecipeFor donutRecipe", donutRecipe)
	if err != nil {
		return err
	}
	h.l.Infof("RecipeFor %#v", donutRecipe)

	// We can't defer calling cancel here because it'll live alongside the stream.
	ctx, cancel := context.WithCancel(context.Background())
	webRTCResponse, err := h.webRTCController.Setup(cancel, donutRecipe, params)
	h.l.Infof("webRTCController.Setup %#v, err=%#v", webRTCResponse, err)
	if err != nil {
		cancel()
		return err
	}
	//tODO: add explan
	h.l.Info("before sleeping")
	time.Sleep(5 * time.Second)
	h.l.Info("after sleeping")
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
			h.l.Infof("onstream %#v", st)
			return h.webRTCController.SendMetadata(webRTCResponse.Data, st)
		},
		OnVideoFrame: func(data []byte, c entities.MediaFrameContext) error {
			// sl[len(sl)-1]
			h.l.Infof("OnVideoFrame %#v < %d > First %#v Last %#v", c, len(data), data[0:7], data[len(data)-7:])
			return h.webRTCController.SendMediaSample(webRTCResponse.Video, data, c)
		},
		OnAudioFrame: func(data []byte, c entities.MediaFrameContext) error {
			h.l.Infof("OnAudioFrame %#v", c)
			return nil
			// return h.webRTCController.SendMediaSample(webRTCResponse.Audio, data, c)
		},
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(*webRTCResponse.LocalSDP)
	h.l.Infof("webRTCResponse %#v", webRTCResponse)
	if err != nil {
		cancel()
		return err
	}

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
