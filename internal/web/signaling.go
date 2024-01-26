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
	"github.com/flavioribeiro/donut/internal/entity"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"go.uber.org/zap"
)

type SignalingHandler struct {
	c             *entity.Config
	l             *zap.Logger
	webrtcSetting *webrtc.SettingEngine
	mediaEngine   *webrtc.MediaEngine
}

func NewSignalingHandler(c *entity.Config, log *zap.Logger, webrtcSetting *webrtc.SettingEngine, mediaEngine *webrtc.MediaEngine) *SignalingHandler {
	return &SignalingHandler{
		c:             c,
		l:             log,
		webrtcSetting: webrtcSetting,
		mediaEngine:   mediaEngine,
	}
}

func (h *SignalingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	SetCORS(w, r)
	if r.Method != http.MethodPost {
		ErrorToHTTP(w, entity.ErrHTTPPostOnly)
		return
	}

	peerConnectionConfiguration := webrtc.Configuration{}
	if !h.c.EnableICEMux {
		peerConnectionConfiguration.ICEServers = []webrtc.ICEServer{
			{
				URLs: h.c.StunServers,
			},
		}
	}

	api := webrtc.NewAPI(
		webrtc.WithSettingEngine(*h.webrtcSetting),
		webrtc.WithMediaEngine(h.mediaEngine),
	)

	peerConnection, err := api.NewPeerConnection(peerConnectionConfiguration)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	offer := entity.ParamsOffer{}
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video", offer.SRTStreamID,
	)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}
	if _, err := peerConnection.AddTrack(videoTrack); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	// Create data channel for metadata transmission
	metadataSender, err := peerConnection.CreateDataChannel("metadata", nil)
	if err != nil {
		ErrorToHTTP(w, err)
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		h.l.Sugar().Infow("ICE Connection State has changed",
			"status", connectionState.String(),
		)
	})

	err = offer.Valid()
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}
	if err = peerConnection.SetRemoteDescription(offer.Offer); err != nil {
		ErrorToHTTP(w, err)
		return
	}

	h.l.Sugar().Infow("Gathering WebRTC Candidates")
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		ErrorToHTTP(w, err)
		return
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		ErrorToHTTP(w, err)
		return
	}
	<-gatherComplete

	h.l.Sugar().Infow("Gathering WebRTC Candidates Complete")

	response, err := json.Marshal(*peerConnection.LocalDescription())
	if err != nil {
		ErrorToHTTP(w, err)
		return
	}

	h.l.Sugar().Infow("Connecting to SRT ",
		"offer", offer,
	)
	srtConnection, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(h.c.SRTConnectionLatencyMS),
			astisrt.WithStreamid(offer.SRTStreamID),
		},

		OnDisconnect: func(c *astisrt.Connection, err error) {
			h.l.Sugar().Fatalw("Disconnected from SRT",
				"error", err,
			)
		},

		Host: offer.SRTHost,
		Port: offer.SRTPort,
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
