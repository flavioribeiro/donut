package probers

import (
	"context"
	"errors"
	"io"

	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/zap"
)

type SrtMpegTs struct {
	c             *entities.Config
	l             *zap.SugaredLogger
	srtController *controllers.SRTController
}

func NewSrtMpegTs(c *entities.Config, l *zap.SugaredLogger, srtController *controllers.SRTController) *SrtMpegTs {
	return &SrtMpegTs{
		c:             c,
		l:             l,
		srtController: srtController,
	}
}

func (c *SrtMpegTs) StreamInfo(req *entities.RequestParams) (map[entities.Codec]entities.Stream, error) {
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srtConnection, err := c.srtController.Connect(cancel, req)
	if err != nil {
		return nil, err
	}
	defer srtConnection.Close()

	streamInfoMap := map[entities.Codec]entities.Stream{}
	inboundMpegTsPacket := make([]byte, c.c.SRTReadBufferSizeBytes)

	probingSize := 120
	// probing mpeg-ts for N packets to find metadata
	c.l.Infow("probing has started")
	go func() {
		for i := 1; i < probingSize; i++ {
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
		c.l.Info("done probing")
		cancel()
	}()
	c.l.Info("probing has starting demuxing")

	mpegTSDemuxer := astits.NewDemuxer(ctx, r)
	for {
		select {
		case <-ctx.Done():
			c.l.Errorw("streaming has stopped")
			return streamInfoMap, nil
		default:
			mpegTSDemuxData, err := mpegTSDemuxer.NextData()

			if err != nil {
				if !errors.Is(err, context.Canceled) {
					c.l.Errorw("failed to demux mpeg-ts",
						"error", err,
					)
					return streamInfoMap, err
				}
				return streamInfoMap, nil
			}

			if mpegTSDemuxData.PMT != nil {

				for _, es := range mpegTSDemuxData.PMT.ElementaryStreams {
					streamInfoMap[mapper.FromMpegTsStreamTypeToCodec(es.StreamType)] = entities.Stream{
						Codec: mapper.FromMpegTsStreamTypeToCodec(es.StreamType),
						Type:  mapper.FromMpegTsStreamTypeToType(es.StreamType),
					}
				}
			}
		}
	}
}
