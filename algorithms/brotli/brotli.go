package brotli

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	"github.com/andybalholm/brotli"
)

func Compress(dst []byte, src []uint64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], u)
	}

	var buf bytes.Buffer
	writer := brotli.NewWriter(&buf)
	writer.Write(uncb)
	writer.Close()

	return buf.Bytes()
}

func CompressFloat(dst []byte, src []float64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		bits := math.Float64bits(u)
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], bits)
	}

	var buf bytes.Buffer
	writer := brotli.NewWriter(&buf)
	writer.Write(uncb)
	writer.Close()

	return buf.Bytes()
}

func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	reader := brotli.NewReader(bytes.NewReader(src))
	var buf bytes.Buffer
	io.Copy(&buf, reader)
	uncb := buf.Bytes()

	for i := 0; i < len(uncb)/8; i++ {
		dst = append(dst, binary.LittleEndian.Uint64(uncb[i*8:(i+1)*8]))
	}
	return dst, nil
}

func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	reader := brotli.NewReader(bytes.NewReader(src))
	var buf bytes.Buffer
	io.Copy(&buf, reader)
	uncb := buf.Bytes()

	for i := 0; i < len(uncb)/8; i++ {
		bits := binary.LittleEndian.Uint64(uncb[i*8 : (i+1)*8])
		dst = append(dst, math.Float64frombits(bits))
	}
	return dst, nil
}
