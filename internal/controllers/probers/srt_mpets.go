package probers

import (
	"context"
	"errors"
	"io"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/flavioribeiro/donut/internal/mapper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SrtMpegTs struct {
	c *entities.Config
	l *zap.SugaredLogger
	m *mapper.Mapper
}

type ResultSrtMpegTs struct {
	fx.Out
	SrtMpegTsProber DonutProber `group:"probers"`
}

// NewSrtMpegTs creates a new SrtMpegTs DonutProber
func NewSrtMpegTs(
	c *entities.Config,
	l *zap.SugaredLogger,
	m *mapper.Mapper,
) ResultSrtMpegTs {
	return ResultSrtMpegTs{
		SrtMpegTsProber: &SrtMpegTs{
			c: c,
			l: l,
			m: m,
		},
	}
}

// Match returns true when the request is for an SrtMpegTs prober
func (c *SrtMpegTs) Match(req *entities.RequestParams) bool {
	if req.SRTHost != "" {
		return true
	}
	return false
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

	srtConnection, err := c.connect(cancel, req)
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
			if errors.Is(ctx.Err(), context.Canceled) {
				c.l.Infow("probing has stopped due cancellation")
				return streamInfoMap, nil
			}
			c.l.Errorw("probing has stopped due errors")
			return streamInfoMap, ctx.Err()
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
			streamInfo[c.m.FromMpegTsStreamTypeToCodec(es.StreamType)] = c.m.FromStreamTypeToEntityStream(es)
		}
	}
	return false, nil
}

// TODO: move to its own component later dup streamer.srt_mpegts, prober.srt_mpegts
func (c *SrtMpegTs) connect(cancel context.CancelFunc, params *entities.RequestParams) (*astisrt.Connection, error) {
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
