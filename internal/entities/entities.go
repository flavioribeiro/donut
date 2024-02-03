package entities

import (
	"context"
	"fmt"

	astisrt "github.com/asticode/go-astisrt/pkg"
	"github.com/pion/webrtc/v3"
)

const (
	MetadataChannelID string = "metadata"
)

type RequestParams struct {
	SRTHost     string
	SRTPort     uint16 `json:",string"`
	SRTStreamID string
	Offer       webrtc.SessionDescription
}

func (p *RequestParams) Valid() error {
	if p == nil {
		return ErrMissingParamsOffer
	}

	if p.SRTHost == "" {
		return ErrMissingSRTHost
	}

	if p.SRTPort == 0 {
		return ErrMissingSRTPort
	}

	if p.SRTStreamID == "" {
		return ErrMissingSRTStreamID
	}

	return nil
}

func (p *RequestParams) String() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("ParamsOffer %v:%v/%v", p.SRTHost, p.SRTPort, p.SRTStreamID)
}

type MessageType string

const (
	MessageTypeMetadata MessageType = "metadata"
)

type Message struct {
	Type    MessageType
	Message string
}

type TrackType string

const (
	H264 TrackType = "h264"
)

type Track struct {
	Type TrackType
}

type Cue struct {
	Type      string
	StartTime int64
	Text      string
}

type StreamParameters struct {
	WebRTCConn    *webrtc.PeerConnection
	Cancel        context.CancelFunc
	Ctx           context.Context
	SRTConnection *astisrt.Connection
	VideoTrack    *webrtc.TrackLocalStaticSample
	MetadataTrack *webrtc.DataChannel
}

type Config struct {
	HTTPPort       int32  `required:"true" default:"8080"`
	HTTPHost       string `required:"true" default:"0.0.0.0"`
	PproffHTTPPort int32  `required:"true" default:"6060"`

	TCPICEPort         int      `required:"true" default:"8081"`
	UDPICEPort         int      `required:"true" default:"8081"`
	ICEReadBufferSize  int      `required:"true" default:"8"`
	ICEExternalIPsDNAT []string `required:"true" default:"127.0.0.1"`
	EnableICEMux       bool     `require:"true" default:"false"`
	StunServers        []string `required:"true" default:"stun:stun4.l.google.com:19302"`

	SRTConnectionLatencyMS int32 `required:"true" default:"300"`
	SRTReadBufferSizeBytes int   `required:"true" default:"1316"`
}
