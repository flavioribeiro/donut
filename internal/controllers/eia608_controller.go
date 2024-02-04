package controllers

import (
	"encoding/json"

	"github.com/asticode/go-astits"
	"github.com/flavioribeiro/donut/internal/entities"
	gocaption "github.com/szatmary/gocaption"
)

type EIA608Reader struct {
	frame gocaption.EIA608Frame
}

func NewEIA608Reader() (r *EIA608Reader) {
	return &EIA608Reader{}
}

func (r *EIA608Reader) Parse(PES *astits.PESData) (string, error) {
	nalus, err := ParseNALUs(PES.Data)
	if err != nil {
		return "", err
	}
	for _, nal := range nalus.Units {
		// ANSI/SCTE 128-1 2020
		// Note that SEI payload is a SEI payloadType of 4 which contains the itu_t_t35_payload_byte for the terminal provider
		if nal.UnitType == entities.SupplementalEnhancementInformation && nal.SEI.PayloadType == 4 {
			// ANSI/SCTE 128-1 2020
			// Caption, AFD and bar data shall be carried in the SEI raw byte sequence payload (RBSP)
			// syntax of the video Elementary Stream.
			cea708Data := nal.RBSPByte[2:] // skip payload type and length bytes
			cea708, err := gocaption.CEA708ToCCData(cea708Data)
			if err != nil {
				return "", err
			}
			for _, c := range cea708 {
				ready, err := r.frame.Decode(c)
				if err != nil {
					panic(err)
				}
				if ready {
					return r.frame.String(), nil
				}
			}
		}
	}
	return "", nil
}

func BuildCaptionsMessage(pts *astits.ClockReference, captions string) (string, error) {
	cue := entities.Cue{
		StartTime: pts.Base,
		Text:      captions,
		Type:      "captions",
	}
	c, err := json.Marshal(cue)
	if err != nil {
		return "", err
	}
	return string(c), nil
}
