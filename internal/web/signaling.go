package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/eia608"
	donutwebrtc "github.com/flavioribeiro/donut/internal/controller/webrtc"
	"github.com/flavioribeiro/donut/internal/entity"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"go.uber.org/zap"
)

type SignalingHandler struct {
	c                *entity.Config
	l                *zap.Logger
	webRTCController *donutwebrtc.WebRTCController
}

func NewSignalingHandler(
	c *entity.Config,
	log *zap.Logger,
	webRTCController *donutwebrtc.WebRTCController,
) *SignalingHandler {
	return &SignalingHandler{
		c:                c,
		l:                log,
		webRTCController: webRTCController,
	}
}

func (h *SignalingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	SetCORS(w, r)
	if r.Method != http.MethodPost {
		ErrorToHTTP(w, entity.ErrHTTPPostOnly)
		return
	}

	browerOffer := entity.ParamsOffer{}
	if err := json.NewDecoder(r.Body).Decode(&browerOffer); err != nil {
		ErrorToHTTP(w, err)
		return
	}
	if err := browerOffer.Valid(); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	if err := h.webRTCController.SetupPeerConnection(); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	// TODO: create tracks according with SRT available streams
	// Create a video track
	videoTrack, err := h.webRTCController.CreateTrack(
		entity.Track{
			Type: entity.H264,
		}, "video", browerOffer.SRTStreamID,
	)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	metadataSender, err := h.webRTCController.CreateDataChannel(entity.MetadataChannelID)
	if err != nil {
		ErrorToHTTP(w, err)
	}

	if err = h.webRTCController.SetRemoteDescription(browerOffer.Offer); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	localDescription, err := h.webRTCController.GatheringWebRTC()
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	response, err := json.Marshal(*localDescription)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	h.l.Sugar().Infow("Connecting to SRT ",
		"offer", browerOffer,
	)
	srtConnection, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(h.c.SRTConnectionLatencyMS),
			astisrt.WithStreamid(browerOffer.SRTStreamID),
		},

		OnDisconnect: func(c *astisrt.Connection, err error) {
			h.l.Sugar().Fatalw("Disconnected from SRT",
				"error", err,
			)
		},

		Host: browerOffer.SRTHost,
		Port: browerOffer.SRTPort,
	})
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}
	h.l.Sugar().Infow("Connected to SRT")

	go srtToWebRTC(srtConnection, videoTrack, metadataSender)

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		ErrorToHTTP(w, err)
		return
	}
}

func srtToWebRTC(srtConnection *astisrt.Connection, videoTrack *webrtc.TrackLocalStaticSample, metadataTrack *webrtc.DataChannel) {
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	defer srtConnection.Close()

	go func() {
		defer srtConnection.Close()
		inboundMpegTsPacket := make([]byte, 1316) // SRT Read Size

		for {
			n, err := srtConnection.Read(inboundMpegTsPacket)
			if err != nil {
				break
			}

			if _, err := w.Write(inboundMpegTsPacket[:n]); err != nil {
				break
			}
		}
	}()

	dmx := astits.NewDemuxer(context.Background(), r)
	eia608Reader := eia608.NewEIA608Reader()
	h264PID := uint16(0)
	for {
		d, err := dmx.NextData()
		if err != nil {
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
				break
			}
			captions, err := eia608Reader.Parse(d.PES)
			if err != nil {
				break
			}
			if captions != "" {
				captionsMsg, err := eia608.BuildCaptionsMessage(d.PES.Header.OptionalHeader.PTS, captions)
				if err != nil {
					break
				}
				metadataTrack.SendText(captionsMsg)
			}
		}
	}

}
