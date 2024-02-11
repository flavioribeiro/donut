package probers

import (
	"context"
	"errors"
	"io"

	astisrt "github.com/asticode/go-astisrt/pkg"
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
	m             *mapper.Mapper
}

func NewSrtMpegTs(c *entities.Config, l *zap.SugaredLogger, srtController *controllers.SRTController, m *mapper.Mapper) *SrtMpegTs {
	return &SrtMpegTs{
		c:             c,
		l:             l,
		srtController: srtController,
		m:             m,
	}
}

// StreamInfo connects to the SRT stream and probe N packets to discovery the media properties.
func (c *SrtMpegTs) StreamInfo(req *entities.RequestParams) (*entities.StreamInfo, error) {
	streamInfoMap, err := c.streamInfoMap(req)
	if err != nil {
		return nil, err
	}

	si := &entities.StreamInfo{}
	for _, v := range streamInfoMap {
		si.Streams = append(si.Streams, v)
	}
	return si, err
}

func (c *SrtMpegTs) streamInfoMap(req *entities.RequestParams) (map[entities.Codec]entities.Stream, error) {
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

	go c.fromSRTToWriterPipe(srtConnection, w, cancel)

	c.l.Info("probing has starting demuxing")

	mpegTSDemuxer := astits.NewDemuxer(ctx, r)
	for {
		select {
		case <-ctx.Done():
			c.l.Errorw("probing has stopped")
			return streamInfoMap, nil
		default:
			stop, err := c.fillStreamInfoFromMpegTS(streamInfoMap, mpegTSDemuxer)
			if stop {
				if err != nil {
					return nil, err
				}
				return streamInfoMap, nil
			}
		}
	}
}

func (c *SrtMpegTs) fromSRTToWriterPipe(srtConnection *astisrt.Connection, w *io.PipeWriter, cancel context.CancelFunc) {
	defer cancel()
	defer w.Close()
	defer srtConnection.Close()

	inboundMpegTsPacket := make([]byte, c.c.SRTReadBufferSizeBytes)
	c.l.Info("probing has started")

	for i := 1; i < c.c.ProbingSize; i++ {
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
}

func (c *SrtMpegTs) fillStreamInfoFromMpegTS(streamInfo map[entities.Codec]entities.Stream, mpegTSDemuxer *astits.Demuxer) (bool, error) {
	mpegTSDemuxData, err := mpegTSDemuxer.NextData()

	if err != nil {
		if !errors.Is(err, context.Canceled) {
			c.l.Errorw("failed to demux mpeg-ts",
				"error", err,
			)
			return true, err
		}
		return true, nil
	}

	if mpegTSDemuxData.PMT != nil {

		for _, es := range mpegTSDemuxData.PMT.ElementaryStreams {
			streamInfo[c.m.FromMpegTsStreamTypeToCodec(es.StreamType)] = c.m.FromStreamTypeToEntityStream(es.StreamType)
		}
	}
	return false, nil
}
