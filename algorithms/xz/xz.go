package xz

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/ulikunitz/xz"
)

func Compress(dst []byte, src []uint64) []byte {
	raw := make([]byte, len(src)*8)
	for i, v := range src {
		binary.LittleEndian.PutUint64(raw[i*8:(i+1)*8], v)
	}
	data, err := compressBytes(raw)
	if err != nil {
		return dst
	}
	return append(dst[:0], data...)
}

func CompressFloat(dst []byte, src []float64) []byte {
	raw := make([]byte, len(src)*8)
	for i, v := range src {
		bits := math.Float64bits(v)
		binary.LittleEndian.PutUint64(raw[i*8:(i+1)*8], bits)
	}
	data, err := compressBytes(raw)
	if err != nil {
		return dst
	}
	return append(dst[:0], data...)
}

func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	raw, err := decompressBytes(src)
	if err != nil {
		return dst, err
	}
	for i := 0; i+8 <= len(raw); i += 8 {
		dst = append(dst, binary.LittleEndian.Uint64(raw[i:i+8]))
	}
	return dst, nil
}

func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	raw, err := decompressBytes(src)
	if err != nil {
		return dst, err
	}
	for i := 0; i+8 <= len(raw); i += 8 {
		bits := binary.LittleEndian.Uint64(raw[i : i+8])
		dst = append(dst, math.Float64frombits(bits))
	}
	return dst, nil
}

func compressBytes(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := xz.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(src); err != nil {
		zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressBytes(src []byte) ([]byte, error) {
	zr, err := xz.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(zr)
}
