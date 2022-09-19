package xevd

/*
#cgo CFLAGS: -I${SRCDIR}/include -I/usr/local/include -I/usr/include -I/usr/local/include/xevdb
#cgo LDFLAGS: -L${SRCDIR} -L/usr/local/lib -L/usr/lib -lxevdb -lm -ldl
#include <stdint.h>
#include <stdlib.h>

#include "xevdb.h"
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"runtime"
	"sync/atomic"
	"unsafe"
)

const (
	defaultMaxBitstreamBufferSize int = 10 * 1024 * 1024
)

type DecodeStreamRecvFunc func(*DecodeImageBuffer)

type BaselineDecoder struct {
	id     unsafe.Pointer // XEVD
	closed int32
}

func (d *BaselineDecoder) Decode(data []byte) (*DecodeImageBuffer, error) {
	if atomic.LoadInt32(&d.closed) == 1 {
		return nil, fmt.Errorf("decoder closed")
	}

	ret := unsafe.Pointer(C.xevdb_decode(
		(C.XEVD)(d.id),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.int(len(data)),
	))
	if ret == nil {
		return nil, fmt.Errorf("xevdb_decode not succeed")
	}

	result := (*C.xevdb_decode_result_t)(ret)
	defer C.xevdb_free_result(result)

	if Ok != int(result.status) {
		return &DecodeImageBuffer{}, nil
	}
	return d.createDecodeImageBuffer(result), nil
}

func (d *BaselineDecoder) createDecodeImageBuffer(result *C.xevdb_decode_result_t) *DecodeImageBuffer {
	yBuf := pool.Get().(*bytes.Buffer)
	yBuf.Reset()
	yBuf.Grow(int(result.size_y))
	yBuf.Write(C.GoBytes(unsafe.Pointer(result.y), result.size_y))

	uBuf := pool.Get().(*bytes.Buffer)
	uBuf.Reset()
	uBuf.Grow(int(result.size_u))
	uBuf.Write(C.GoBytes(unsafe.Pointer(result.u), result.size_u))

	vBuf := pool.Get().(*bytes.Buffer)
	vBuf.Reset()
	vBuf.Grow(int(result.size_v))
	vBuf.Write(C.GoBytes(unsafe.Pointer(result.v), result.size_v))

	releaseFunc := createReleasePoolFunc(yBuf, uBuf, vBuf)

	depth := int(result.bit_depth)
	y := convertBitDepth(yBuf.Bytes(), depth)
	u := convertBitDepth(uBuf.Bytes(), depth)
	v := convertBitDepth(vBuf.Bytes(), depth)

	img := &image.YCbCr{
		Rect:           image.Rect(0, 0, int(result.width), int(result.height)),
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Y:              y,
		Cb:             u,
		Cr:             v,
		YStride:        int(result.stride_y),
		CStride:        int(result.stride_u),
	}
	buf := &DecodeImageBuffer{
		NALUnit:    NALUnitType(result.nalu_type),
		Slice:      SliceType(result.slice_type),
		ColorSpace: ColorFormatType(result.color_format),
		Img:        img,
		closeFunc:  releaseFunc,
		closed:     0,
	}
	buf.setFinalizer()
	return buf
}

func (d *BaselineDecoder) DecodeStream(r io.Reader, fn DecodeStreamRecvFunc) error {
	naluLengthByte := make([]byte, NALUnitLengthByteSize)
	naluDataBuf := pool.Get().(*bytes.Buffer)
	defer pool.Put(naluDataBuf)

	for {

		if _, err := r.Read(naluLengthByte); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		naluLength := binary.BigEndian.Uint32(naluLengthByte)
		naluDataBuf.Reset()
		if _, err := io.CopyN(naluDataBuf, r, int64(naluLength)); err != nil {
			return fmt.Errorf("Unexpected read error(read-size:%d) %w", naluLength, err)
		}

		buf, err := d.Decode(naluDataBuf.Bytes())
		if err != nil {
			return err
		}

		if buf.HasData() != true {
			continue
		}
		fn(buf)
	}
}

func (d *BaselineDecoder) Close() {
	if atomic.CompareAndSwapInt32(&d.closed, 0, 1) {
		runtime.SetFinalizer(d, nil)

		C.xevdb_free_xevd(
			(C.XEVD)(d.id),
		)
	}
}

func finalizeBaselineDecoder(d *BaselineDecoder) {
	d.Close()
}

func CreateDefaultBaselineDecoder() (*BaselineDecoder, error) {
	return CreateBaselineDecoder(
		runtime.NumCPU(),
		defaultMaxBitstreamBufferSize,
	)
}

func CreateBaselineDecoder(threads, maxBitstreamBufferSize int) (*BaselineDecoder, error) {
	id := unsafe.Pointer(C.xevdb_create(
		C.int(threads),
	))
	if id == nil {
		return nil, fmt.Errorf("failed to call xevdb_create()")
	}
	d := &BaselineDecoder{
		id:     id,
		closed: 0,
	}
	runtime.SetFinalizer(d, finalizeBaselineDecoder)
	return d, nil
}
