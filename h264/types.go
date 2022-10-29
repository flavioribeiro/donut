package h264

type NALUs struct {
	Units []NAL
}

// Rec. ITU-T H.264 (08/2021) p.43
type NAL struct {
	RefIDC      byte
	UnitType    NALUnitType
	RBSPByte    []byte
	HeaderBytes []byte
	SEI
}

type SEI struct {
	PayloadType int
	PayloadSize int
}

type NALUnitType byte

const (
	// Rec. ITU-T H.264 (08/2021) p.65
	Unspecified0                                            = NALUnitType(0)  //	Unspecified
	CodedSliceNonIDRPicture                                 = NALUnitType(1)  //	Coded slice of a non-IDR picture
	CodedSliceDataPartitionA                                = NALUnitType(2)  //	Coded slice data partition A
	CodedSliceDataPartitionB                                = NALUnitType(3)  //	Coded slice data partition B
	CodedSliceDataPartitionC                                = NALUnitType(4)  //	Coded slice data partition C
	CodedSliceIDRPicture                                    = NALUnitType(5)  //	Coded slice of an IDR picture
	SupplementalEnhancementInformation                      = NALUnitType(6)  //	Supplemental enhancement information (SEI)
	SequenceParameterSet                                    = NALUnitType(7)  //	Sequence parameter set
	PictureParameterSet                                     = NALUnitType(8)  //	Picture parameter set
	AccessUnitDelimiter                                     = NALUnitType(9)  //	Access unit delimiter
	EndOfSequence                                           = NALUnitType(10) //	End of sequence
	EndOfStream                                             = NALUnitType(11) //	End of stream
	FillerData                                              = NALUnitType(12) //	Filler data
	SequenceParameterSetExtension                           = NALUnitType(13) //	Sequence parameter set extension
	PrefixNALUnit                                           = NALUnitType(14) //	Prefix NAL unit
	SubsetSequenceParameterSet                              = NALUnitType(15) //	Subset sequence parameter set
	DepthParameterSet                                       = NALUnitType(16) //	Depth parameter set
	Reserved17                                              = NALUnitType(17) //	Reserved
	Reserved18                                              = NALUnitType(18) //	Reserved
	CodedSliceAuxiliaryCodedPictureWithoutPartitioning      = NALUnitType(19) //	Coded slice of an auxiliary coded  picture without partitioning
	CodedSliceExtension                                     = NALUnitType(20) //	Coded slice extension
	CodedSliceExtensionDepthViewComponentOr3DAVCTextureView = NALUnitType(21) //	Coded slice extension for a depth view component or a 3D-AVC texture view component
	Reserved22                                              = NALUnitType(22) //	Reserved
	Reserved23                                              = NALUnitType(23) //	Reserved
	Unspecified24                                           = NALUnitType(24) //	Unspecified
	Unspecified25                                           = NALUnitType(25) //	Unspecified
	Unspecified26                                           = NALUnitType(26) //	Unspecified
	Unspecified27                                           = NALUnitType(27) //	Unspecified
	Unspecified28                                           = NALUnitType(28) //	Unspecified
	Unspecified29                                           = NALUnitType(29) //	Unspecified
	Unspecified30                                           = NALUnitType(30) //	Unspecified
	Unspecified31                                           = NALUnitType(31) //	Unspecified

)
