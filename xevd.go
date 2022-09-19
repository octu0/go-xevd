package xevd

import (
	"bytes"
	"image"
	"runtime"
	"sync"
	"sync/atomic"
)

type ReturnCode int

const (
	NoMoreFrames             ReturnCode = 205
	OutNotAvailable                     = 204
	FrameDimensionChanged               = 203
	FrameDelayed                        = 202
	ErrWarnCRCIgnored                   = 200
	Ok                                  = 0
	Err                                 = -1
	ErrInvalidArgument                  = -101
	ErrOutOfMemory                      = -102
	ErrReachedMax                       = -103
	ErrUnsupported                      = -104
	ErrUnexpected                       = -105
	ErrUnsupportedColorSpace            = -201
	ErrMalformedBitstream               = -202
	ErrThreadAllocation                 = -203
	ErrBadCRC                           = -300
	ErrUnknown                          = -32767
)

func (rc ReturnCode) Error() string {
	switch rc {
	// Succeed
	case Ok:
		return "XEVD_OK"
	case NoMoreFrames:
		return "XEVD_OK_NO_MORE_FRM"
	case OutNotAvailable:
		return "XEVD_OK_OUT_NOT_AVAILABLE"
	case FrameDimensionChanged:
		return "XEVD_OK_DIM_CHANGED"
	case FrameDelayed:
		return "XEVD_OK_FRM_DELAYED"
	case ErrWarnCRCIgnored:
		return "XEVD_WARN_CRC_IGNORED"
	// Failed
	case Err:
		return "XEVD_ERR"
	case ErrInvalidArgument:
		return "XEVD_ERR_INVALID_ARGUMENT"
	case ErrOutOfMemory:
		return "XEVD_ERR_OUT_OF_MEMORY"
	case ErrReachedMax:
		return "XEVD_ERR_REACHED_MAX"
	case ErrUnsupported:
		return "XEVD_ERR_UNSUPPORTED"
	case ErrUnexpected:
		return "XEVD_ERR_UNEXPECTED"
	case ErrUnsupportedColorSpace:
		return "XEVD_ERR_UNSUPPORTED_COLORSPACE"
	case ErrMalformedBitstream:
		return "XEVD_ERR_MALFORMED_BITSTREAM"
	case ErrThreadAllocation:
		return "XEVD_ERR_THREAD_ALLOCATION"
	case ErrBadCRC:
		return "XEVD_ERR_BAD_CRC"
	}
	return "XEVD_ERR_UNKNOWN"
}

func Succeed(rc ReturnCode) bool {
	if Ok <= rc {
		return true
	}
	return false
}

func Failed(rc ReturnCode) bool {
	if rc < Ok {
		return true
	}
	return false
}

type ColorFormatType uint8

const (
	ColorFormatUnknown   = 0
	ColorFormatYCbCr400  = 10 // Y onlu
	ColorFormatYCbCr420  = 11 // YCbCr 420
	ColorFormatYCbCr422  = 12 // YCbCr 422 narrow chroma
	ColorFormatYCbCr444  = 13 // YCbCr 444
	ColorFormatYCbCr422N = ColorFormatYCbCr422
	ColorFormatYCbCr422W = 18 // YCbCr 422 wide chroma
)

func (c ColorFormatType) String() string {
	switch c {
	case ColorFormatYCbCr400:
		return "YCbCr400"
	case ColorFormatYCbCr420:
		return "YCbCr420"
	case ColorFormatYCbCr422:
		return "YCbCr422"
	case ColorFormatYCbCr444:
		return "YCbCr444"
	case ColorFormatYCbCr422W:
		return "YCbCr422W"
	}
	return "Unknown"
}

type ConfigType uint16

const (
	ConfigSetUsePicSignature ConfigType = 301
	ConfigGetCodecBitDepth              = 401
	ConfigGetWith                       = 402
	ConfigGetHeight                     = 403
	ConfigGetCodedWith                  = 404
	ConfigGetCodedHeight                = 405
	ConfigGetCodedSpace                 = 406
	ConfigGetMaxConfigDelay             = 407
)

type NALUnitType uint8

const (
	NALUnitNonIDR NALUnitType = 0
	NALUnitIDR                = 1
	NALUnitSPS                = 24
	NALUnitPPS                = 25
	NALUnitAPS                = 26
	NALUnitFD                 = 27
	NALUnitSEI                = 28
)

const NALUnitLengthByteSize = 4

func (n NALUnitType) String() string {
	switch n {
	case NALUnitNonIDR:
		return "NonIDR"
	case NALUnitIDR:
		return "IDR"
	case NALUnitSPS:
		return "SPS"
	case NALUnitPPS:
		return "PPS"
	case NALUnitAPS:
		return "APS"
	case NALUnitFD:
		return "FD"
	case NALUnitSEI:
		return "SEI"
	default:
		return "Unknown"
	}
}

type SliceType int8

const (
	SliceUnknown SliceType = -1
	SliceB                 = 0
	SliceP                 = 1
	SliceI                 = 2
)

func (s SliceType) String() string {
	switch s {
	case SliceB:
		return "B"
	case SliceP:
		return "P"
	case SliceI:
		return "I"
	default:
		return "Unknown"
	}
}

var (
	pool = &sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}
)

func createReleasePoolFunc(yBuf, uBuf, vBuf *bytes.Buffer) func() {
	return func() {
		pool.Put(yBuf)
		pool.Put(uBuf)
		pool.Put(vBuf)
	}
}

type DecodeImageBuffer struct {
	NALUnit    NALUnitType
	Slice      SliceType
	ColorSpace ColorFormatType
	Img        *image.YCbCr
	closeFunc  func()
	closed     int32
}

func (b *DecodeImageBuffer) HasData() bool {
	if b.Img != nil {
		return true
	}
	return false
}

func (b *DecodeImageBuffer) setFinalizer() {
	runtime.SetFinalizer(b, func(me *DecodeImageBuffer) {
		me.Close()
	})
}

func (b *DecodeImageBuffer) Close() {
	if atomic.CompareAndSwapInt32(&b.closed, 0, 1) {
		runtime.SetFinalizer(b, nil)

		if b.closeFunc != nil {
			b.closeFunc()
		}
	}
}

// yuv 8bit (passthrough)
func convertBitDepth8(data []byte) []byte {
	return data
}

// yuv 10bit to 8bit
func convertBitDepth10(data []byte) []byte {
	size := len(data)
	pos := 0
	for i := 0; i < size; i += 2 {
		b8 := (data[i] >> 2) | (data[i+1] << 6)
		data[pos] = b8
		pos += 1
	}
	return data[0:pos:pos]
}

func convertBitDepth(data []byte, bitDepth int) []byte {
	if 8 == bitDepth {
		return convertBitDepth8(data)
	}
	if 10 == bitDepth {
		return convertBitDepth10(data)
	}
	// otherwize
	return data
}
