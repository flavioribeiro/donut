package entities

import (
	"context"
	"fmt"
	"time"

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

type Codec string
type MediaType string

const (
	UnknownCodec Codec = "unknownCodec"
	H264         Codec = "h264"
	H265         Codec = "h265"
	VP9          Codec = "vp9"
	AV1          Codec = "av1"
	AAC          Codec = "aac"
	Opus         Codec = "opus"
)

const (
	UnknownType MediaType = "unknownMediaType"
	VideoType   MediaType = "video"
	AudioType   MediaType = "audio"
)

type Stream struct {
	Codec Codec
	Type  MediaType
	Id    uint16
	Index uint16
}

type MediaFrameContext struct {
	// DTS decoding timestamp
	DTS int
	// PTS presentation timestamp
	PTS int
	// Media frame duration
	Duration time.Duration
}

type StreamInfo struct {
	Streams []Stream
}

func (s *StreamInfo) VideoStreams() []Stream {
	var result []Stream
	for _, s := range s.Streams {
		if s.Type == VideoType {
			result = append(result, s)
		}
	}
	return result
}

func (s *StreamInfo) AudioStreams() []Stream {
	var result []Stream
	for _, s := range s.Streams {
		if s.Type == AudioType {
			result = append(result, s)
		}
	}
	return result
}

type Cue struct {
	Type      string
	StartTime int64
	Text      string
}

type DonutParameters struct {
	Cancel context.CancelFunc
	Ctx    context.Context

	StreamID     string // ie: live001, channel01
	StreamFormat string // ie: flv, mpegts
	StreamURL    string // ie: srt://host:9080, rtmp://host:4991

	TranscodeVideoCodec Codec // ie: vp8
	TranscodeAudioCodec Codec // ie: opus

	OnClose      func()
	OnError      func(err error)
	OnStream     func(st *Stream)
	OnVideoFrame func(data []byte, c MediaFrameContext) error
	OnAudioFrame func(data []byte, c MediaFrameContext) error
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
	StunServers        []string `required:"true" default:"stun:stun.l.google.com:19302,stun:stun1.l.google.com:19302,stun:stun2.l.google.com:19302,stun:stun4.l.google.com:19302"`

	SRTConnectionLatencyMS int32 `required:"true" default:"300"`
	// MPEG-TS consists of single units of 188 bytes. Multiplying 188*7 we get 1316,
	// which is the maximum product of 188 that is less than MTU 1500 (188*8=1504)
	// ref https://github.com/Haivision/srt/blob/master/docs/features/live-streaming.md#transmitting-mpeg-ts-binary-protocol-over-srt
	SRTReadBufferSizeBytes int `required:"true" default:"1316"`

	ProbingSize int `required:"true" default:"120"`
}
