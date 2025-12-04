package huffmanLib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/icza/huffman/hufio"
)

// CompressBytes 压缩任意字节流，保留与其他接口一致的标记格式
func CompressBytes(dst []byte, src []byte) []byte {
	return append(dst, buildHuffmanBlock(src)...)
}

func buildHuffmanBlock(src []byte) []byte {
	var buf bytes.Buffer
	w := hufio.NewWriter(&buf)
	if _, err := w.Write(src); err != nil {
		return wrapUncompressed(src)
	}
	if err := w.Close(); err != nil {
		return wrapUncompressed(src)
	}
	compressed := buf.Bytes()
	if len(compressed) == 0 || len(compressed) >= len(src) {
		return wrapUncompressed(src)
	}
	res := make([]byte, 1+len(compressed))
	res[0] = 1
	copy(res[1:], compressed)
	return res
}

func wrapUncompressed(src []byte) []byte {
	res := make([]byte, 1+len(src))
	res[0] = 0
	copy(res[1:], src)
	return res
}

// DecompressBytes 解压到字节流
func DecompressBytes(dst []byte, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return dst, fmt.Errorf("empty input")
	}
	flag := src[0]
	data := src[1:]
	var (
		uncb []byte
		err  error
	)
	if flag == 1 {
		r := hufio.NewReader(bytes.NewReader(data))
		uncb, err = io.ReadAll(r)
		if err != nil {
			return dst, err
		}
	} else {
		uncb = data
	}
	return append(dst, uncb...), nil
}

// Compress 压缩 uint64 数组，行为类似 ans.Compress：当压缩无效时返回未压缩的数据并在首字节标记为0，压缩成功时标记为1
func Compress(dst []byte, src []uint64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], u)
	}

	return CompressBytes(dst, uncb)
}

// CompressFloat 压缩 float64 数组（使用与 Compress 相同的封装）
func CompressFloat(dst []byte, src []float64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		bits := math.Float64bits(u)
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], bits)
	}

	return CompressBytes(dst, uncb)
}

// Decompress 解压缩到 uint64 数组，支持 Compress 写入的标记格式
func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	uncb, err := DecompressBytes(nil, src)
	if err != nil {
		return dst, err
	}

	for i := 0; i < len(uncb)/8; i++ {
		dst = append(dst, binary.LittleEndian.Uint64(uncb[i*8:(i+1)*8]))
	}
	return dst, nil
}

// DecompressFloat 解压缩到 float64 数组
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	if len(src) == 0 {
		return dst, nil
	}
	uncb, err := DecompressBytes(nil, src)
	if err != nil {
		return dst, err
	}

	for i := 0; i < len(uncb)/8; i++ {
		bits := binary.LittleEndian.Uint64(uncb[i*8 : (i+1)*8])
		dst = append(dst, math.Float64frombits(bits))
	}
	return dst, nil
}
