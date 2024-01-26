package entity

import (
	"errors"
	"fmt"

	"github.com/pion/webrtc/v3"
)

type Media struct{}

type ParamsOffer struct {
	SRTHost     string
	SRTPort     uint16 `json:",string"`
	SRTStreamID string
	Offer       webrtc.SessionDescription
}

func (p *ParamsOffer) Valid() error {
	if p == nil {
		return errors.New("ParamsOffer must not be nil")
	}

	if p.SRTHost == "" {
		return errors.New("SRTHost must not be nil")
	}

	if p.SRTPort == 0 {
		return errors.New("SRTPort must be valid")
	}

	if p.SRTStreamID == "" {
		return errors.New("SRTStreamID must not be empty")
	}

	return nil
}

func (p *ParamsOffer) String() string {
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
