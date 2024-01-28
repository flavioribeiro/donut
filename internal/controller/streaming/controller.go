package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/eia608"
	"github.com/flavioribeiro/donut/internal/entity"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"go.uber.org/zap"
)

type StreamingController struct {
	c *entity.Config
	l *zap.Logger
}

func NewStreamingController(c *entity.Config, l *zap.Logger) *StreamingController {
	return &StreamingController{
		c: c,
		l: l,
	}
}

func (c *StreamingController) Stream(_ context.Context, srtConnection *astisrt.Connection, videoTrack *webrtc.TrackLocalStaticSample, metadataTrack *webrtc.DataChannel) {
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	defer srtConnection.Close()

	c.l.Sugar().Infow("start streaming")

	// reading from SRT writing into pipe writer
	go func() {
		defer srtConnection.Close()
		// TODO: pick the proper transport? is it possible to get rtp instead?
		inboundMpegTsPacket := make([]byte, c.c.SRTReadBufferSizeBytes) // SRT Read Size

		for {
			n, err := srtConnection.Read(inboundMpegTsPacket)
			if err != nil {
				c.l.Sugar().Errorw("str conn failed to read mpeg ts",
					"error", err,
				)
				break
			}

			if _, err := w.Write(inboundMpegTsPacket[:n]); err != nil {
				c.l.Sugar().Errorw("failed to write mpeg ts in the pipe",
					"error", err,
				)
				break
			}
		}
	}()

	dmx := astits.NewDemuxer(context.Background(), r)
	eia608Reader := eia608.NewEIA608Reader()
	h264PID := uint16(0)

	// reading from reader pipe
	for {
		d, err := dmx.NextData()
		if err != nil {
			c.l.Sugar().Errorw("failed to demux mpeg ts",
				"error", err,
			)
			break
		}

		if d.PMT != nil {
			for _, es := range d.PMT.ElementaryStreams {
				msg, _ := json.Marshal(entity.Message{
					Type:    entity.MessageTypeMetadata,
					Message: es.StreamType.String(),
				})
				metadataTrack.SendText(string(msg))
				if es.StreamType == astits.StreamTypeH264Video {
					h264PID = es.ElementaryPID
				}
			}

			for _, d := range d.PMT.ProgramDescriptors {
				if d.MaximumBitrate != nil {
					bitrateInMbitsPerSecond := float32(d.MaximumBitrate.Bitrate) / float32(125000)
					msg, _ := json.Marshal(entity.Message{
						Type:    entity.MessageTypeMetadata,
						Message: fmt.Sprintf("Bitrate %.2fMbps", bitrateInMbitsPerSecond),
					})
					metadataTrack.SendText(string(msg))
				}
			}
		}

		if d.PID == h264PID && d.PES != nil {
			if err = videoTrack.WriteSample(media.Sample{Data: d.PES.Data, Duration: time.Second / 30}); err != nil {
				c.l.Sugar().Errorw("failed to write a sample mpeg ts to web rtc",
					"error", err,
				)
				break
			}
			captions, err := eia608Reader.Parse(d.PES)
			if err != nil {
				c.l.Sugar().Errorw("failed to parse eia 608",
					"error", err,
				)
				break
			}
			if captions != "" {
				captionsMsg, err := eia608.BuildCaptionsMessage(d.PES.Header.OptionalHeader.PTS, captions)
				if err != nil {
					c.l.Sugar().Errorw("failed to build captions message",
						"error", err,
					)
					break
				}
				metadataTrack.SendText(captionsMsg)
			}
		}
	}
}
