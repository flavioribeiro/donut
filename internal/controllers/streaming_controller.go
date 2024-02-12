package controllers

import (
	"context"
	"errors"
	"io"
	"time"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/pion/webrtc/v3/pkg/media"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type StreamingControllerParams struct {
	fx.In
	C *entities.Config
	L *zap.SugaredLogger

	EIA608Middleware     entities.StreamMiddleware `name:"eia608"`
	StreamInfoMiddleware entities.StreamMiddleware `name:"stream_info"`
}

type StreamingController struct {
	c *entities.Config
	l *zap.SugaredLogger

	middlewares []entities.StreamMiddleware
}

func NewStreamingController(sp StreamingControllerParams) *StreamingController {
	middlewares := []entities.StreamMiddleware{sp.EIA608Middleware, sp.StreamInfoMiddleware}

	return &StreamingController{
		c:           sp.C,
		l:           sp.L,
		middlewares: middlewares,
	}
}

func (c *StreamingController) Stream(sp *entities.StreamParameters) {
	r, w := io.Pipe()

	defer r.Close()
	defer w.Close()
	defer sp.SRTConnection.Close()
	defer sp.WebRTCConn.Close()
	defer sp.Cancel()

	// TODO: pick the proper transport? is it possible to get rtp instead?
	go c.readFromSRTIntoWriterPipe(sp.SRTConnection, w)

	// reading from reader pipe to the mpeg-ts demuxer
	mpegTSDemuxer := astits.NewDemuxer(sp.Ctx, r)

	c.l.Infow("streaming has started")

	for {
		select {
		case <-sp.Ctx.Done():
			if errors.Is(sp.Ctx.Err(), context.Canceled) {
				c.l.Infow("streaming has stopped due cancellation")
				return
			}
			c.l.Errorw("streaming has stopped due errors",
				"error", sp.Ctx.Err(),
			)
			return
		default:
			// fetching mpeg-ts data
			// ref https://tsduck.io/download/docs/mpegts-introduction.pdf
			mpegTSDemuxData, err := mpegTSDemuxer.NextData()
			if err != nil {
				c.l.Errorw("failed to demux mpeg-ts",
					"error", err,
				)
				return
			}

			// writing mpeg-ts video to webrtc channels
			for _, v := range sp.StreamInfo.VideoStreams() {
				if v.Id != mpegTSDemuxData.PID {
					continue
				}

				if err := sp.VideoTrack.WriteSample(media.Sample{Data: mpegTSDemuxData.PES.Data, Duration: time.Second / 30}); err != nil {
					c.l.Errorw("failed to write an mpeg-ts to web rtc",
						"error", err,
					)
					return
				}
			}
			if err != nil {
				c.l.Errorw("failed to write an mpeg-ts to web rtc",
					"error", err,
				)
				return
			}

			// calling all registered middlewares
			for _, m := range c.middlewares {
				err = m.Act(mpegTSDemuxData, sp)
				if err != nil {
					c.l.Errorw("middleware error",
						"error", err,
					)
				}
			}
		}
	}
}

func (c *StreamingController) readFromSRTIntoWriterPipe(srtConnection *astisrt.Connection, w *io.PipeWriter) {
	defer srtConnection.Close()

	inboundMpegTsPacket := make([]byte, c.c.SRTReadBufferSizeBytes)

	for {
		n, err := srtConnection.Read(inboundMpegTsPacket)
		if err != nil {
			c.l.Errorw("str conn failed to write data to buffer",
				"error", err,
			)
			break
		}

		if _, err := w.Write(inboundMpegTsPacket[:n]); err != nil {
			c.l.Errorw("failed to write mpeg-ts into the pipe",
				"error", err,
			)
			break
		}
	}
}
