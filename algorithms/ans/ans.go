package ans

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/klauspost/compress/fse"
)

func Compress(dst []byte, src []uint64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], u)
	}

	compressed, err := fse.Compress(uncb, nil)
	if err != nil {
		// FSE 压缩失败，添加标志位 0 表示未压缩，然后返回原始数据
		result := make([]byte, 1+len(uncb))
		result[0] = 0 // 未压缩标志
		copy(result[1:], uncb)
		return result
	}

	// 如果压缩后的数据比原始数据还大，返回原始数据
	if len(compressed) == 0 || len(compressed) >= len(uncb) {
		result := make([]byte, 1+len(uncb))
		result[0] = 0 // 未压缩标志
		copy(result[1:], uncb)
		return result
	}

	// 添加标志位 1 表示已压缩
	result := make([]byte, 1+len(compressed))
	result[0] = 1 // 已压缩标志
	copy(result[1:], compressed)
	return result
}

func CompressFloat(dst []byte, src []float64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		bits := math.Float64bits(u)
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], bits)
	}

	compressed, _ := fse.Compress(uncb, nil)
	return compressed
}

func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	if len(src) == 0 {
		return dst, fmt.Errorf("empty input")
	}

	// 读取标志位
	compressed := src[0]
	data := src[1:]

	var uncb []byte
	var err error

	if compressed == 1 {
		// 数据已压缩，需要解压
		uncb, err = fse.Decompress(data, nil)
		if err != nil {
			return dst, err
		}
	} else {
		// 数据未压缩，直接使用
		uncb = data
	}

	for i := 0; i < len(uncb)/8; i++ {
		dst = append(dst, binary.LittleEndian.Uint64(uncb[i*8:(i+1)*8]))
	}
	return dst, nil
}

func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	uncb, err := fse.Decompress(src, nil)
	if err != nil {
		return dst, err
	}

	for i := 0; i < len(uncb)/8; i++ {
		bits := binary.LittleEndian.Uint64(uncb[i*8 : (i+1)*8])
		dst = append(dst, math.Float64frombits(bits))
	}
	return dst, nil
}
