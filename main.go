//go:build !js
// +build !js

package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"donut/h264"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/asticode/go-astits"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	gocaption "github.com/szatmary/gocaption"
)

var (
	//go:embed index.html
	indexHTML string
)

func assertSignalingCorrect(SRTHost, SRTPort, SRTStreamID string) (int, error) {
	switch {
	case SRTHost == "":
		return 0, errors.New("SRTHost must not be nil")
	case SRTPort == "":
		return 0, errors.New("SRTPort must not be empty")
	case SRTStreamID == "":
		return 0, errors.New("SRTStreamID must not be empty")
	}

	return strconv.Atoi(SRTPort)
}

func errorToHTTP(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}

func srtToWebRTC(srtConnection *astisrt.Connection, videoTrack *webrtc.TrackLocalStaticSample) {
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
	eia608 := &gocaption.EIA608Frame{}
	h264PID := uint16(0)
	for {
		d, err := dmx.NextData()
		if err != nil {
			break
		}

		if d.PMT != nil {
			for _, es := range d.PMT.ElementaryStreams {
				if es.StreamType == astits.StreamTypeH264Video {
					h264PID = es.ElementaryPID
				}
			}
		}

		if d.PID == h264PID && d.PES != nil {
			if err = videoTrack.WriteSample(media.Sample{Data: d.PES.Data, Duration: time.Second / 30}); err != nil {
				break
			}
			parseAndPrint608Captions(eia608, d.PES)
		}
	}

}

func parseAndPrint608Captions(eia608Reader *gocaption.EIA608Frame, PES *astits.PESData) {
	nalus, err := h264.ParseNALUs(PES.Data)
	if err != nil {
		panic(err)
	}
	for _, nal := range nalus.Units {
		// ANSI/SCTE 128-1 2020
		// Note that SEI payload is a SEI payloadType of 4 which contains the itu_t_t35_payload_byte for the terminal provider
		if nal.UnitType == h264.SupplementalEnhancementInformation && nal.SEI.PayloadType == 4 {
			// ANSI/SCTE 128-1 2020
			// Caption, AFD and bar data shall be carried in the SEI raw byte sequence payload (RBSP)
			// syntax of the video Elementary Stream.
			buffer := bytes.NewBuffer(nal.RBSPByte)
			buffer.Next(2) // skip payload type and length
			test708, err := gocaption.CEA708ToCCData(buffer.Bytes())
			if err != nil {
				panic(err)
			}
			for _, c := range test708 {
				ready, err := eia608Reader.Decode(c)
				if err != nil {
					panic(err)
				}
				if ready {
					fmt.Printf("captions: %s\n", eia608Reader.String())
				}
			}
		}
	}
}

func doSignaling(w http.ResponseWriter, r *http.Request) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun4.l.google.com:19302",
				},
			},
		},
	})
	if err != nil {
		errorToHTTP(w, err)
		return
	}

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "spongebob")
	if err != nil {
		errorToHTTP(w, err)
		return
	}
	if _, err := peerConnection.AddTrack(videoTrack); err != nil {
		errorToHTTP(w, err)
		return
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	offer := struct {
		SRTHost, SRTPort, SRTStreamID string
		Offer                         webrtc.SessionDescription
	}{"", "", "", webrtc.SessionDescription{}}

	if err = json.NewDecoder(r.Body).Decode(&offer); err != nil {
		errorToHTTP(w, err)
		return
	}

	srtPort, err := assertSignalingCorrect(offer.SRTHost, offer.SRTPort, offer.SRTStreamID)
	if err != nil {
		errorToHTTP(w, err)
		return
	}

	if err = peerConnection.SetRemoteDescription(offer.Offer); err != nil {
		errorToHTTP(w, err)
		return
	}

	log.Println("Gathering WebRTC Candidates")
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		errorToHTTP(w, err)
		return
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		errorToHTTP(w, err)
		return
	}
	<-gatherComplete
	log.Println("Gathering WebRTC Candidates Complete")

	response, err := json.Marshal(*peerConnection.LocalDescription())
	if err != nil {
		return
	}

	log.Println("Connecting to SRT")
	srtConnection, err := astisrt.Dial(astisrt.DialOptions{
		ConnectionOptions: []astisrt.ConnectionOption{
			astisrt.WithLatency(300),
			astisrt.WithStreamid(offer.SRTStreamID),
		},

		// Callback when the connection is disconnected
		OnDisconnect: func(c *astisrt.Connection, err error) { panic("Disconnected from SRT") },

		Host: offer.SRTHost,
		Port: uint16(srtPort),
	})
	if err != nil {
		errorToHTTP(w, err)
		return
	}
	log.Println("Connected to SRT")

	go srtToWebRTC(srtConnection, videoTrack)

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		errorToHTTP(w, err)
		return
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(indexHTML))
	})
	http.HandleFunc("/doSignaling", doSignaling)

	log.Println("Open http://localhost:8080 to access this demo")
	panic(http.ListenAndServe(":8080", nil))
}
