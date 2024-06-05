package streammiddlewares

import (
	"encoding/json"

	"github.com/flavioribeiro/donut/internal/controllers"
	"github.com/flavioribeiro/donut/internal/entities"
	gocaption "github.com/szatmary/gocaption"
)

type eia608Reader struct {
	frame gocaption.EIA608Frame
}

func newEIA608Reader() (r *eia608Reader) {
	return &eia608Reader{}
}

func (r *eia608Reader) parse(data []byte) (string, error) {
	nalus, err := controllers.ParseNALUs(data)
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

// TODO: port to mappers
func (r *eia608Reader) buildCaptionsMessage(pts int64, captions string) (string, error) {
	cue := entities.Cue{
		StartTime: pts,
		Text:      captions,
		Type:      "captions",
	}
	c, err := json.Marshal(cue)
	if err != nil {
		return "", err
	}
	return string(c), nil
}
