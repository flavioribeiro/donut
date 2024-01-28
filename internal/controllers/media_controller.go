package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/eia608"
	"github.com/flavioribeiro/donut/internal/entities"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"go.uber.org/zap"
)

type StreamingController struct {
	c *entities.Config
	l *zap.Logger
}

func NewStreamingController(c *entities.Config, l *zap.Logger) *StreamingController {
	return &StreamingController{
		c: c,
		l: l,
	}
}

func (c *StreamingController) Stream(srtConnection *astisrt.Connection, videoTrack *webrtc.TrackLocalStaticSample, metadataTrack *webrtc.DataChannel) {
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	defer srtConnection.Close()

	c.l.Sugar().Infow("start streaming")

	// TODO: pick the proper transport? is it possible to get rtp instead?
	go c.readFromSRTIntoWriterPipe(srtConnection, w)

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
			h264PID = c.captureMediaInfoAndSendToWebRTC(d, metadataTrack, h264PID)
			c.captureBitrateAndSendToWebRTC(d, metadataTrack)
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

func (*StreamingController) captureBitrateAndSendToWebRTC(d *astits.DemuxerData, metadataTrack *webrtc.DataChannel) {
	for _, d := range d.PMT.ProgramDescriptors {
		if d.MaximumBitrate != nil {
			bitrateInMbitsPerSecond := float32(d.MaximumBitrate.Bitrate) / float32(125000)
			msg, _ := json.Marshal(entities.Message{
				Type:    entities.MessageTypeMetadata,
				Message: fmt.Sprintf("Bitrate %.2fMbps", bitrateInMbitsPerSecond),
			})
			metadataTrack.SendText(string(msg))
		}
	}
}

func (*StreamingController) captureMediaInfoAndSendToWebRTC(d *astits.DemuxerData, metadataTrack *webrtc.DataChannel, h264PID uint16) uint16 {
	for _, es := range d.PMT.ElementaryStreams {

		msg, _ := json.Marshal(entities.Message{
			Type:    entities.MessageTypeMetadata,
			Message: es.StreamType.String(),
		})
		metadataTrack.SendText(string(msg))

		if es.StreamType == astits.StreamTypeH264Video {
			h264PID = es.ElementaryPID
		}
	}
	return h264PID
}

func (c *StreamingController) readFromSRTIntoWriterPipe(srtConnection *astisrt.Connection, w *io.PipeWriter) {
	defer srtConnection.Close()

	inboundMpegTsPacket := make([]byte, c.c.SRTReadBufferSizeBytes)

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
}
