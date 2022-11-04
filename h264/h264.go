package h264

import (
	"bytes"
	"fmt"
)

func ParseNALUs(data []byte) (NALUs, error) {
	var nalus NALUs

	rawNALUs := bytes.Split(data, []byte{0x00, 0x00, 0x01})

	for _, rawNALU := range rawNALUs[1:] {
		nal, err := ParseNAL(rawNALU)
		if err != nil {
			return NALUs{}, err

		}
		nalus.Units = append(nalus.Units, nal)
	}

	return nalus, nil
}

func ParseNAL(data []byte) (NAL, error) {
	index := 0
	n := NAL{}
	if data[index]>>7&0x01 != 0 {
		return NAL{}, fmt.Errorf("forbidden_zero_bit is not 0")
	}
	n.RefIDC = (data[index] >> 5) & 0x03
	n.UnitType = NALUnitType(data[index] & 0x1f)
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

func (n *NAL) ParseRBSP() error {
	switch n.UnitType {
	case SupplementalEnhancementInformation:
		err := n.parseSEI()
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *NAL) parseSEI() error {
	numBits := 0
	byteOffset := 0
	n.SEI.PayloadType = 0
	n.SEI.PayloadSize = 0
	nextBits := n.RBSPByte[byteOffset]

	for {
		if nextBits == 0xff {
			n.PayloadType += 255
			numBits += 8
			byteOffset += numBits / 8
			numBits = numBits % 8
			nextBits = n.RBSPByte[byteOffset]
			continue
		}
		break
	}

	n.PayloadType += int(nextBits)
	numBits += 8
	byteOffset += numBits / 8
	numBits = numBits % 8
	nextBits = n.RBSPByte[byteOffset]

	// read size
	for {
		if nextBits == 0xff {
			n.PayloadSize += 255
			numBits += 8
			byteOffset += numBits / 8
			numBits = numBits % 8
			nextBits = n.RBSPByte[byteOffset]
			continue
		}
		break
	}

	n.PayloadSize += int(nextBits)
	numBits += 8
	byteOffset += numBits / 8

	return nil
}
