package simple8b

import (
	"encoding/binary"
	"fmt"

	"github.com/influxdata/influxdb/pkg/encoding/simple8b"
)

// func Compress(dst []byte, src []uint64) []byte {
// 	simple8bdst, _ := simple8b.EncodeAll(src)
// 	fmt.Printf("simple8b.EncodeAll returned %d words\n", len(simple8bdst))
// 	for _, val := range simple8bdst {
// 		dst = common.Append64(dst, val)
// 	}
// 	return dst
// }

// func Decompress(dst []uint64, src []byte) ([]uint64, error) {

// 	var compressedByte []uint64
// 	for i := 0; i < len(src); i += 8 {
// 		compressedByte = append(compressedByte, binary.LittleEndian.Uint64(src[i:i+8]))
// 	}

//		num, err := simple8b.DecodeAll(dst, compressedByte)
//		fmt.Println("simple8b.Decompress:", num)
//		if err != nil {
//			fmt.Println("simple8b Decompress failed:", err)
//			return dst, err
//		}
//		return dst, nil
//	}

// 安全版本的压缩函数 - 不修改输入数组
func Compress(dst []byte, src []uint64) []byte {
	// 创建输入数组的副本，避免修改原始数组
	srcCopy := make([]uint64, len(src))
	copy(srcCopy, src)

	words, err := simple8b.EncodeAll(srcCopy)
	if err != nil {
		panic(fmt.Sprintf("simple8b.EncodeAll failed: %v", err))
	}
	// fmt.Printf("simple8b.Compress: encoded %d values into %d words\n",
	// 	len(src), len(words))
	for _, w := range words {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, w)
		dst = append(dst, buf...)
	}
	return dst
}

func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	if len(src)%8 != 0 {
		return dst, fmt.Errorf("invalid src length: %d", len(src))
	}

	words := make([]uint64, len(src)/8)
	for i := 0; i < len(src); i += 8 {
		words[i/8] = binary.LittleEndian.Uint64(src[i : i+8])
	}

	// 使用临时缓冲区，避免直接修改dst
	temp := make([]uint64, len(dst))
	n, err := simple8b.DecodeAll(temp, words)
	if err != nil {
		return dst, fmt.Errorf("decode failed: %w", err)
	}

	if n > len(dst) {
		n = len(dst)
	}
	copy(dst, temp[:n])

	// fmt.Printf("simple8b.Decompress: decoded %d values\n", n)

	return dst[:n], nil
}
