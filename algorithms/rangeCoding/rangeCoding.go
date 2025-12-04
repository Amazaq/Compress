package rangeCoding

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	headerFreqCount = 256
	headerSize      = 4 + headerFreqCount*4 // original-length + 256 uint32 frequencies

	codeValueBits = 32
	topValue      = (uint64(1) << codeValueBits) - 1
	firstQtr      = topValue/4 + 1
	half          = firstQtr * 2
	thirdQtr      = firstQtr * 3
)

// RangeEncoder 使用标准算术编码（range coding）实现
type RangeEncoder struct {
	low        uint64
	high       uint64
	pending    uint32
	bitBuffer  byte
	bitsFilled uint8
	out        []byte
}

// NewRangeEncoder 创建新的范围编码器
func NewRangeEncoder() *RangeEncoder {
	return &RangeEncoder{
		low:  0,
		high: topValue,
		out:  make([]byte, 0, 1024),
	}
}

// Encode 根据累计频率编码符号
func (e *RangeEncoder) Encode(cumLow, cumHigh, total uint32) {
	if cumHigh <= cumLow || cumHigh > total || total == 0 {
		panic("rangeCoding: invalid cumulative frequencies")
	}
	rangeVal := e.high - e.low + 1
	e.high = e.low + (rangeVal*uint64(cumHigh))/uint64(total) - 1
	e.low = e.low + (rangeVal*uint64(cumLow))/uint64(total)

	for {
		switch {
		case e.high < half:
			e.outputBit(0)
		case e.low >= half:
			e.outputBit(1)
			e.low -= half
			e.high -= half
		case e.low >= firstQtr && e.high < thirdQtr:
			e.pending++
			e.low -= firstQtr
			e.high -= firstQtr
		default:
			return
		}
		e.low <<= 1
		e.high = (e.high << 1) | 1
	}
}

func (e *RangeEncoder) outputBit(bit uint32) {
	e.writeBit(bit)
	for e.pending > 0 {
		e.writeBit(bit ^ 1)
		e.pending--
	}
}

func (e *RangeEncoder) writeBit(bit uint32) {
	e.bitBuffer = (e.bitBuffer << 1) | byte(bit&1)
	e.bitsFilled++
	if e.bitsFilled == 8 {
		e.out = append(e.out, e.bitBuffer)
		e.bitBuffer = 0
		e.bitsFilled = 0
	}
}

// Finish 写出剩余范围
func (e *RangeEncoder) Finish() []byte {
	e.pending++
	if e.low < firstQtr {
		e.outputBit(0)
	} else {
		e.outputBit(1)
	}
	if e.bitsFilled > 0 {
		e.out = append(e.out, e.bitBuffer<<(8-e.bitsFilled))
	}
	return e.out
}

// RangeDecoder 范围解码器
type RangeDecoder struct {
	low  uint64
	high uint64
	code uint64
	in   []byte
	bitN int
}

// NewRangeDecoder 创建新的范围解码器
func NewRangeDecoder(data []byte) *RangeDecoder {
	d := &RangeDecoder{
		low:  0,
		high: topValue,
		in:   data,
		bitN: 0,
	}
	for i := 0; i < codeValueBits; i++ {
		d.code = (d.code << 1) | uint64(d.readBit())
	}
	return d
}

func (d *RangeDecoder) readBit() uint32 {
	byteIndex := d.bitN / 8
	if byteIndex >= len(d.in) {
		d.bitN++
		return 0
	}
	bitIndex := 7 - (d.bitN % 8)
	bit := (d.in[byteIndex] >> bitIndex) & 1
	d.bitN++
	return uint32(bit)
}

// Decode 根据累计频率表解码一个符号
func (d *RangeDecoder) Decode(cumFreq []uint32, total uint32) byte {
	if total == 0 {
		panic("rangeCoding: invalid total frequency")
	}
	rangeVal := d.high - d.low + 1
	value := uint32(((d.code-d.low+1)*uint64(total) - 1) / rangeVal)
	symbol := sort.Search(headerFreqCount, func(i int) bool {
		return cumFreq[i+1] > value
	})
	lowCount := cumFreq[symbol]
	highCount := cumFreq[symbol+1]
	d.high = d.low + (rangeVal*uint64(highCount))/uint64(total) - 1
	d.low = d.low + (rangeVal*uint64(lowCount))/uint64(total)

	for {
		switch {
		case d.high < half:
			// no-op
		case d.low >= half:
			d.low -= half
			d.high -= half
			d.code -= half
		case d.low >= firstQtr && d.high < thirdQtr:
			d.low -= firstQtr
			d.high -= firstQtr
			d.code -= firstQtr
		default:
			return byte(symbol)
		}
		d.low <<= 1
		d.high = (d.high << 1) | 1
		d.code = (d.code << 1) | uint64(d.readBit())
	}
}

func buildFrequencyTable(data []byte) ([]uint32, []uint32, uint32) {
	freq := make([]uint32, headerFreqCount)
	for i := 0; i < headerFreqCount; i++ {
		freq[i] = 1 // 平滑，避免 0 概率
	}
	for _, b := range data {
		freq[int(b)]++
	}
	cum := make([]uint32, headerFreqCount+1)
	for i := 0; i < headerFreqCount; i++ {
		cum[i+1] = cum[i] + freq[i]
	}
	total := cum[headerFreqCount]
	return freq, cum, total
}

func encodeBytes(data []byte) []byte {
	freq, cum, total := buildFrequencyTable(data)
	enc := NewRangeEncoder()
	for _, b := range data {
		idx := int(b)
		enc.Encode(cum[idx], cum[idx+1], total)
	}
	payload := enc.Finish()
	result := make([]byte, 0, headerSize+len(payload))
	result = binary.LittleEndian.AppendUint32(result, uint32(len(data)))
	for _, f := range freq {
		result = binary.LittleEndian.AppendUint32(result, f)
	}
	result = append(result, payload...)
	return result
}

func decodeBytes(src []byte) ([]byte, error) {
	if len(src) < headerSize {
		return nil, fmt.Errorf("rangeCoding: input too short (%d bytes)", len(src))
	}
	originalLen := binary.LittleEndian.Uint32(src[:4])
	offset := 4
	freq := make([]uint32, headerFreqCount)
	for i := 0; i < headerFreqCount; i++ {
		freq[i] = binary.LittleEndian.Uint32(src[offset : offset+4])
		offset += 4
	}
	cum := make([]uint32, headerFreqCount+1)
	for i := 0; i < headerFreqCount; i++ {
		cum[i+1] = cum[i] + freq[i]
	}
	total := cum[headerFreqCount]
	if total == 0 {
		return nil, errors.New("rangeCoding: invalid frequency table")
	}
	decoder := NewRangeDecoder(src[offset:])
	out := make([]byte, 0, originalLen)
	for uint32(len(out)) < originalLen {
		b := decoder.Decode(cum, total)
		out = append(out, b)
	}
	return out, nil
}

// CompressBytes 直接压缩字节流
func CompressBytes(dst []byte, src []byte) []byte {
	return append(dst, encodeBytes(src)...)
}

// DecompressBytes 还原字节流
func DecompressBytes(dst []byte, src []byte) ([]byte, error) {
	decoded, err := decodeBytes(src)
	if err != nil {
		return dst, err
	}
	return append(dst, decoded...), nil
}

// Compress 压缩 uint64 数组
func Compress(dst []byte, src []uint64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], u)
	}
	return CompressBytes(dst, uncb)
}

// CompressFloat 压缩 float64 数组
func CompressFloat(dst []byte, src []float64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		bits := math.Float64bits(u)
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], bits)
	}
	return CompressBytes(dst, uncb)
}

// Decompress 解压缩到 uint64 数组
func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	uncb, err := DecompressBytes(nil, src)
	if err != nil {
		return dst, err
	}
	if len(uncb)%8 != 0 {
		return dst, fmt.Errorf("rangeCoding: invalid payload size %d", len(uncb))
	}
	for i := 0; i < len(uncb); i += 8 {
		dst = append(dst, binary.LittleEndian.Uint64(uncb[i:i+8]))
	}
	return dst, nil
}

// DecompressFloat 解压缩到 float64 数组
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	uncb, err := DecompressBytes(nil, src)
	if err != nil {
		return dst, err
	}
	if len(uncb)%8 != 0 {
		return dst, fmt.Errorf("rangeCoding: invalid payload size %d", len(uncb))
	}
	for i := 0; i < len(uncb); i += 8 {
		bits := binary.LittleEndian.Uint64(uncb[i : i+8])
		dst = append(dst, math.Float64frombits(bits))
	}
	return dst, nil
}
