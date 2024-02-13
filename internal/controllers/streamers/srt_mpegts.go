package streamers

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

type SRTMpegTSStreamer struct {
	c *entities.Config
	l *zap.SugaredLogger

	middlewares []entities.StreamMiddleware
}

type SRTMpegTSStreamerParams struct {
	fx.In
	C *entities.Config
	L *zap.SugaredLogger

	Middlewares []entities.StreamMiddleware `group:"middlewares"`
}

type ResultSRTMpegTSStreamer struct {
	fx.Out
	SRTMpegTSStreamer DonutStreamer `group:"streamers"`
}

func NewSRTMpegTSStreamer(p SRTMpegTSStreamerParams) ResultSRTMpegTSStreamer {
	return ResultSRTMpegTSStreamer{
		SRTMpegTSStreamer: &SRTMpegTSStreamer{
			c:           p.C,
			l:           p.L,
			middlewares: p.Middlewares,
		},
	}
}

func (c *SRTMpegTSStreamer) Match(req *entities.RequestParams) bool {
	if req.SRTHost != "" {
		return true
	}
	return false
}

func (c *SRTMpegTSStreamer) Stream(sp *entities.StreamParameters) {
	srtConnection, err := c.connect(sp.Cancel, sp.RequestParams)
	if err != nil {
		c.l.Errorw("streaming has stopped due errors",
			"error", sp.Ctx.Err(),
		)
		return
	}
	r, w := io.Pipe()

	defer r.Close()
	defer w.Close()
	defer srtConnection.Close()
	defer sp.WebRTCConn.Close()
	defer sp.Cancel()

	go c.readFromSRTIntoWriterPipe(srtConnection, w)

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
			for _, v := range sp.ServerStreamInfo.VideoStreams() {
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

func (c *SRTMpegTSStreamer) readFromSRTIntoWriterPipe(srtConnection *astisrt.Connection, w *io.PipeWriter) {
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

// TODO: move to its own component later dup streamer.srt_mpegts, prober.srt_mpegts
func (c *SRTMpegTSStreamer) connect(cancel context.CancelFunc, params *entities.RequestParams) (*astisrt.Connection, error) {
	c.l.Info("trying to connect srt")

	if err := params.Valid(); err != nil {
		return nil, err
	}

	c.l.Infow("Connecting to SRT ",
		"offer", params.String(),
	)

	conn, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(c.c.SRTConnectionLatencyMS),
			astisrt.WithStreamid(params.SRTStreamID),
			astisrt.WithCongestion("live"),
			astisrt.WithTranstype(astisrt.Transtype(astisrt.TranstypeLive)),
		},

		OnDisconnect: func(conn *astisrt.Connection, err error) {
			c.l.Infow("Canceling SRT",
				"error", err,
			)
			cancel()
		},

		Host: params.SRTHost,
		Port: params.SRTPort,
	})
	if err != nil {
		c.l.Errorw("failed to connect srt",
			"error", err,
		)
		return nil, err
	}
	c.l.Infow("Connected to SRT")
	return conn, nil
}
