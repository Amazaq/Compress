package xor

import (
	"encoding/binary"
	"fmt"
	"math"
	"myalgo/algorithms/chimp"
	"myalgo/algorithms/simple8b"
)

// splitFloat 将浮点数分解为尾数部分和指数部分
// 例如: 3.25 = 11.01(二进制) = 0.1101 * 2^2
// 返回: 尾数部分(0.8125), 指数部分(2)
func splitFloat(f float64) (mantissa float64, exponent uint64) {
	// 处理特殊情况
	if f == 0 {
		return 0, 0
	}

	// 获取符号
	sign := 1.0
	if f < 0 {
		sign = -1.0
		f = -f
	}

	// 如果数字已经小于1，不需要移位
	if f < 1.0 {
		return sign * f, 0
	}

	// 计算需要右移的位数（即指数）
	exponent = 0
	mantissa = f

	// 将数字归一化到[0.5, 1)区间
	for mantissa >= 1.0 {
		mantissa /= 2.0
		exponent++
	}

	// 恢复符号
	mantissa *= sign

	return mantissa, exponent
}

// splitFloatArray 对数组中的每个浮点数进行分解
// 返回两个数组：尾数数组和指数数组
func splitFloatArray(src []float64) (mantissas []float64, exponents []uint64) {
	mantissas = make([]float64, len(src))
	exponents = make([]uint64, len(src))

	for i, f := range src {
		mantissas[i], exponents[i] = splitFloat(f)
	}

	return mantissas, exponents
}

// CompressFloat 压缩浮点数数组
// 使用分离压缩策略：尾数用CHIMP压缩，指数用Simple8b压缩
func CompressFloat(dst []byte, src []float64) []byte {
	// 处理空数组
	if len(src) == 0 {
		if dst == nil {
			dst = make([]byte, 0, 8)
		}
		lengthBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(lengthBytes, 0)
		return append(dst, lengthBytes...)
	}

	// 分解浮点数数组
	mantissas, exponents := splitFloatArray(src)
	// for i := 0; i < 10; i++ {
	// 	fmt.Printf("Split:%f, %d\n", mantissas[i], exponents[i])
	// }
	// 准备压缩
	if dst == nil {
		// 预估压缩后的大小
		estimatedSize := len(src)*8 + 1024 // 额外空间用于元数据
		dst = make([]byte, 0, estimatedSize)
	}

	// 写入原始数组长度（用于解压时恢复）
	lengthBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(lengthBytes, uint64(len(src)))
	dst = append(dst, lengthBytes...)

	// 压缩尾数部分（使用CHIMP算法）
	var mantissaCompressed []byte
	mantissaCompressed = chimp.CompressFloat(mantissaCompressed, mantissas)
	// 写入尾数压缩数据的长度
	mantissaLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(mantissaLenBytes, uint32(len(mantissaCompressed)))
	dst = append(dst, mantissaLenBytes...)
	// 写入尾数压缩数据
	dst = append(dst, mantissaCompressed...)

	// 压缩指数部分（使用Simple8b算法）
	var exponentCompressed []byte
	exponentCompressed = simple8b.Compress(exponentCompressed, exponents)
	// 写入指数压缩数据的长度
	exponentLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(exponentLenBytes, uint32(len(exponentCompressed)))
	dst = append(dst, exponentLenBytes...)
	// 写入指数压缩数据
	dst = append(dst, exponentCompressed...)

	return dst
}

// DecompressFloat 解压缩浮点数数组
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	if len(src) < 8 {
		return nil, fmt.Errorf("invalid compressed data: too short")
	}

	offset := 0

	// 读取原始数组长度
	originalLength := binary.LittleEndian.Uint64(src[offset : offset+8])
	offset += 8

	// 处理空数组情况
	if originalLength == 0 {
		return []float64{}, nil
	}

	// 读取尾数压缩数据长度
	if offset+4 > len(src) {
		return nil, fmt.Errorf("invalid compressed data: cannot read mantissa length")
	}
	mantissaLen := binary.LittleEndian.Uint32(src[offset : offset+4])
	offset += 4

	// 读取并解压尾数数据
	if offset+int(mantissaLen) > len(src) {
		return nil, fmt.Errorf("invalid compressed data: mantissa data truncated")
	}
	mantissaData := src[offset : offset+int(mantissaLen)]
	offset += int(mantissaLen)
	var mantissas []float64
	mantissas, err := chimp.DecompressFloat(mantissas, mantissaData)
	// for i := 0; i < 10; i++ {
	// 	fmt.Printf("Decompress Mantissas: %f\n", mantissas[i])
	// }
	if err != nil {
		return nil, fmt.Errorf("failed to decompress mantissas: %w", err)
	}

	// 读取指数压缩数据长度
	if offset+4 > len(src) {
		return nil, fmt.Errorf("invalid compressed data: cannot read exponent length")
	}
	exponentLen := binary.LittleEndian.Uint32(src[offset : offset+4])
	offset += 4

	// 读取并解压指数数据
	if offset+int(exponentLen) > len(src) {
		return nil, fmt.Errorf("invalid compressed data: exponent data truncated")
	}
	exponentData := src[offset : offset+int(exponentLen)]

	// 检查是否有指数数据
	if len(exponentData) == 0 {
		return nil, fmt.Errorf("empty exponent data")
	}

	// 为simple8b解压缩预分配空间
	exponentUint64 := make([]uint64, originalLength+240)
	exponentUint64, err = simple8b.Decompress(exponentUint64, exponentData)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress exponents: %w", err)
	}

	// 检查数组长度是否匹配
	if len(mantissas) != int(originalLength) || len(exponentUint64) != int(originalLength) {
		return nil, fmt.Errorf("decompressed array length mismatch: expected %d, got mantissas %d, exponents %d",
			originalLength, len(mantissas), len(exponentUint64))
	}

	// 重建原始浮点数
	result := make([]float64, originalLength)
	for i := 0; i < int(originalLength); i++ {
		// 重建: 原始值 = mantissa * 2^exponent
		if exponentUint64[i] == 0 {
			result[i] = mantissas[i]
		} else {
			result[i] = mantissas[i] * math.Pow(2, float64(exponentUint64[i]))
		}
	}

	return result, nil
}
