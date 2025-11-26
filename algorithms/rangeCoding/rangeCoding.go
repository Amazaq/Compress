package rangeCoding

import (
	"encoding/binary"
	"math"
)

const (
	topValue    = uint32(1 << 24)
	bottomValue = uint32(1 << 16)
	rangeWidth  = uint32(1 << 8)
)

// RangeEncoder 范围编码器
type RangeEncoder struct {
	low      uint32
	rangeVal uint32
	out      []byte
}

// NewRangeEncoder 创建新的范围编码器
func NewRangeEncoder() *RangeEncoder {
	return &RangeEncoder{
		low:      0,
		rangeVal: 0xFFFFFFFF,
		out:      make([]byte, 0),
	}
}

// Encode 编码一个字节
func (e *RangeEncoder) Encode(symbol byte) {
	e.rangeVal /= 256
	e.low += uint32(symbol) * e.rangeVal

	for e.rangeVal < bottomValue {
		e.out = append(e.out, byte(e.low>>24))
		e.low <<= 8
		e.rangeVal <<= 8
	}
}

// Finish 完成编码
func (e *RangeEncoder) Finish() []byte {
	for i := 0; i < 4; i++ {
		e.out = append(e.out, byte(e.low>>24))
		e.low <<= 8
	}
	return e.out
}

// RangeDecoder 范围解码器
type RangeDecoder struct {
	low      uint32
	rangeVal uint32
	code     uint32
	in       []byte
	pos      int
}

// NewRangeDecoder 创建新的范围解码器
func NewRangeDecoder(data []byte) *RangeDecoder {
	d := &RangeDecoder{
		low:      0,
		rangeVal: 0xFFFFFFFF,
		code:     0,
		in:       data,
		pos:      0,
	}

	for i := 0; i < 4; i++ {
		if d.pos < len(d.in) {
			d.code = (d.code << 8) | uint32(d.in[d.pos])
			d.pos++
		}
	}

	return d
}

// Decode 解码一个字节
func (d *RangeDecoder) Decode() byte {
	d.rangeVal /= 256
	symbol := byte((d.code - d.low) / d.rangeVal)
	d.low += uint32(symbol) * d.rangeVal

	for d.rangeVal < bottomValue {
		d.low <<= 8
		d.rangeVal <<= 8
		if d.pos < len(d.in) {
			d.code = (d.code << 8) | uint32(d.in[d.pos])
			d.pos++
		} else {
			d.code <<= 8
		}
	}

	return symbol
}

// Compress 压缩 uint64 数组
func Compress(dst []byte, src []uint64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], u)
	}

	encoder := NewRangeEncoder()
	for _, b := range uncb {
		encoder.Encode(b)
	}

	return encoder.Finish()
}

// CompressFloat 压缩 float64 数组
func CompressFloat(dst []byte, src []float64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		bits := math.Float64bits(u)
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], bits)
	}

	encoder := NewRangeEncoder()
	for _, b := range uncb {
		encoder.Encode(b)
	}

	return encoder.Finish()
}

// Decompress 解压缩到 uint64 数组
func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	decoder := NewRangeDecoder(src)

	// 估算原始数据大小（这里需要存储在压缩数据的头部，暂时简化处理）
	// 实际应用中应该在压缩时记录原始大小
	uncb := make([]byte, 0)
	for decoder.pos < len(src) {
		b := decoder.Decode()
		uncb = append(uncb, b)
	}

	for i := 0; i < len(uncb)/8; i++ {
		dst = append(dst, binary.LittleEndian.Uint64(uncb[i*8:(i+1)*8]))
	}
	return dst, nil
}

// DecompressFloat 解压缩到 float64 数组
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	decoder := NewRangeDecoder(src)

	// 估算原始数据大小（这里需要存储在压缩数据的头部，暂时简化处理）
	uncb := make([]byte, 0)
	for decoder.pos < len(src) {
		b := decoder.Decode()
		uncb = append(uncb, b)
	}

	for i := 0; i < len(uncb)/8; i++ {
		bits := binary.LittleEndian.Uint64(uncb[i*8 : (i+1)*8])
		dst = append(dst, math.Float64frombits(bits))
	}
	return dst, nil
}
