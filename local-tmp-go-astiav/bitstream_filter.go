package astiav

//#cgo pkg-config: libavcodec
//#include <libavcodec/bsf.h>
//#include <stdlib.h>
import "C"
import (
	"errors"
	"unsafe"
)

// https://github.com/FFmpeg/FFmpeg/blob/release/5.1/libavcodec/bsf.h#L68
type BSFContext struct {
	c *C.struct_AVBSFContext
}

func newBSFContextFromC(c *C.struct_AVBSFContext) *BSFContext {
	if c == nil {
		return nil
	}
	bsfCtx := &BSFContext{c: c}
	classers.set(bsfCtx)
	return bsfCtx
}

var _ Classer = (*BSFContext)(nil)

func AllocBitStreamContext(f *BitStreamFilter) (*BSFContext, error) {
	if f == nil {
		return nil, errors.New("astiav: bit stream filter must not be nil")
	}

	var bsfCtx *C.struct_AVBSFContext
	if err := newError(C.av_bsf_alloc(f.c, &bsfCtx)); err != nil {
		return nil, err
	}

	return newBSFContextFromC(bsfCtx), nil
}

func (bsfCtx *BSFContext) Class() *Class {
	return newClassFromC(unsafe.Pointer(bsfCtx.c))
}

func (bsfCtx *BSFContext) Init() error {
	return newError(C.av_bsf_init(bsfCtx.c))
}

func (bsfCtx *BSFContext) SendPacket(p *Packet) error {
	if p == nil {
		return errors.New("astiav: packet must not be nil")
	}
	return newError(C.av_bsf_send_packet(bsfCtx.c, p.c))
}

func (bsfCtx *BSFContext) ReceivePacket(p *Packet) error {
	if p == nil {
		return errors.New("astiav: packet must not be nil")
	}
	return newError(C.av_bsf_receive_packet(bsfCtx.c, p.c))
}

func (bsfCtx *BSFContext) Free() {
	classers.del(bsfCtx)
	C.av_bsf_free(&bsfCtx.c)
}

func (bsfCtx *BSFContext) TimeBaseIn() Rational {
	return newRationalFromC(bsfCtx.c.time_base_in)
}

func (bsfCtx *BSFContext) SetTimeBaseIn(r Rational) {
	bsfCtx.c.time_base_in = r.c
}

func (bsfCtx *BSFContext) TimeBaseOut() Rational {
	return newRationalFromC(bsfCtx.c.time_base_out)
}

func (bsfCtx *BSFContext) SetTimeBaseOut(r Rational) {
	bsfCtx.c.time_base_out = r.c
}

func (bsfCtx *BSFContext) CodecParametersIn() *CodecParameters {
	return newCodecParametersFromC(bsfCtx.c.par_in)
}

func (bsfCtx *BSFContext) SetCodecParametersIn(cp *CodecParameters) {
	bsfCtx.c.par_in = cp.c
}

func (bsfCtx *BSFContext) CodecParametersOut() *CodecParameters {
	return newCodecParametersFromC(bsfCtx.c.par_out)
}

func (bsfCtx *BSFContext) SetCodecParametersOut(cp *CodecParameters) {
	bsfCtx.c.par_out = cp.c
}

// https://github.com/FFmpeg/FFmpeg/blob/release/5.1/libavcodec/bsf.h#L111
type BitStreamFilter struct {
	c *C.struct_AVBitStreamFilter
}

func newBitStreamFilterFromC(c *C.struct_AVBitStreamFilter) *BitStreamFilter {
	if c == nil {
		return nil
	}
	cc := &BitStreamFilter{c: c}
	classers.set(cc)
	return cc
}

func FindBitStreamFilterByName(n string) *BitStreamFilter {
	cn := C.CString(n)
	defer C.free(unsafe.Pointer(cn))
	return newBitStreamFilterFromC(C.av_bsf_get_by_name(cn))
}

func (bsf *BitStreamFilter) Class() *Class {
	return newClassFromC(unsafe.Pointer(bsf.c))
}

func (bsf *BitStreamFilter) Name() string {
	return C.GoString(bsf.c.name)
}

// TODO: learn how to work with c arrays
// func (bsf *BitStreamFilter) Codecs() (cids []*CodecID) {
// 	scs := (*[(math.MaxInt32 - 1) / unsafe.Sizeof((C.enum_AVCodecID)(nil))](C.enum_AVCodecID))(unsafe.Pointer(bsf.c.codec_ids))
// 	for i := 0; i < fc.NbStreams(); i++ {
// 		cids = append(cids, CodecID(scs[i]))
// 	}
// 	return cids
// }
