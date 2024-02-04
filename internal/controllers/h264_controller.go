package controllers

import (
	"bytes"
	"fmt"

	"github.com/flavioribeiro/donut/internal/entities"
)

func ParseNALUs(data []byte) (entities.NALUs, error) {
	var nalus entities.NALUs

	rawNALUs := bytes.Split(data, []byte{0x00, 0x00, 0x01})

	for _, rawNALU := range rawNALUs[1:] {
		nal, err := ParseNAL(rawNALU)
		if err != nil {
			return entities.NALUs{}, err

		}
		nalus.Units = append(nalus.Units, nal)
	}

	return nalus, nil
}

func ParseNAL(data []byte) (entities.NAL, error) {
	index := 0
	n := entities.NAL{}
	if data[index]>>7&0x01 != 0 {
		return entities.NAL{}, fmt.Errorf("forbidden_zero_bit is not 0")
	}
	n.RefIDC = (data[index] >> 5) & 0x03
	n.UnitType = entities.NALUnitType(data[index] & 0x1f)
	numBytesInRBSP := 0
	nalUnitHeaderBytes := 1
	n.HeaderBytes = data[:nalUnitHeaderBytes]

	index += nalUnitHeaderBytes

	n.RBSPByte = make([]byte, 0, 16)
	i := 0
	for i = index; i < len(data); i++ {
		if (i+2) < len(data) && (data[i] == 0x00 && data[i+1] == 0x00 && data[i+2] == 0x03) {
			n.RBSPByte = append(n.RBSPByte, data[i], data[i+1])
			i += 2
			numBytesInRBSP += 2
			// 0x03
		} else {
			n.RBSPByte = append(n.RBSPByte, data[i])
			numBytesInRBSP++
		}
	}
	index += numBytesInRBSP

	n.ParseRBSP()

	return n, nil
}
