package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/asticode/go-astits"
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

func (c *StreamingController) Stream(sp entities.StreamParameters) {
	r, w := io.Pipe()

	defer r.Close()
	defer w.Close()
	defer sp.SRTConnection.Close()
	defer sp.WebRTCConn.Close()
	defer sp.Cancel()

	c.l.Sugar().Infow("start streaming")

	// TODO: pick the proper transport? is it possible to get rtp instead?
	go c.readFromSRTIntoWriterPipe(sp.SRTConnection, w)

	// reading from reader pipe into mpeg-ts demuxer
	mpegTSDemuxer := astits.NewDemuxer(sp.Ctx, r)
	eia608Reader := NewEIA608Reader()
	h264PID := uint16(0)

	for {
		select {
		case <-sp.Ctx.Done():
			c.l.Sugar().Errorw("stream was cancelled")
			return
		default:
			// ref https://tsduck.io/download/docs/mpegts-introduction.pdf
			mpegTSDemuxData, err := mpegTSDemuxer.NextData()
			if err != nil {
				c.l.Sugar().Errorw("failed to demux mpeg-ts",
					"error", err,
				)
				return
			}

			if mpegTSDemuxData.PMT != nil {
				// writing mpeg-ts meida metadata to the metadata webrtc channel
				h264PID = c.captureMediaInfoAndSendToWebRTC(mpegTSDemuxData, sp.MetadataTrack, h264PID)
				c.captureBitrateAndSendToWebRTC(mpegTSDemuxData, sp.MetadataTrack)
			}

			if mpegTSDemuxData.PID == h264PID && mpegTSDemuxData.PES != nil {
				// writing video from mpeg-ts into webrtc
				if err = sp.VideoTrack.WriteSample(media.Sample{Data: mpegTSDemuxData.PES.Data, Duration: time.Second / 30}); err != nil {
					c.l.Sugar().Errorw("failed to write a sample mpeg-ts to web rtc",
						"error", err,
					)
					return
				}
				captions, err := eia608Reader.Parse(mpegTSDemuxData.PES)
				if err != nil {
					c.l.Sugar().Errorw("failed to parse eia 608",
						"error", err,
					)
					return
				}
				if captions != "" {
					captionsMsg, err := BuildCaptionsMessage(mpegTSDemuxData.PES.Header.OptionalHeader.PTS, captions)
					if err != nil {
						c.l.Sugar().Errorw("failed to build captions message",
							"error", err,
						)
						return
					}
					// writing metadata to the metadata webrtc channel
					sp.MetadataTrack.SendText(captionsMsg)
				}
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
