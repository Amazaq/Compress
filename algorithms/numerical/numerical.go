package numerical

import (
	"encoding/binary"
	"fmt"
	"math"

	// "myalgo/algorithms/ans"
	// "myalgo/algorithms/rangeCoding"
	// "myalgo/algorithms/huffmanLib"
	// "myalgo/algorithms/lz77"

	brotlicodec "myalgo/algorithms/brotli"
	lz4codec "myalgo/algorithms/lz4"
	snappycodec "myalgo/algorithms/snappy"
	xzcodec "myalgo/algorithms/xz"

	"github.com/valyala/gozstd"
)

type (
	compressWithConstraintsFunc   func([]byte, []float64, *NumericalConstraints) []byte
	decompressWithConstraintsFunc func([]float64, []byte, *NumericalConstraints) ([]float64, error)
	uint64BackendCompressor       func([]byte, []uint64) []byte
	uint64BackendDecompressor     func([]uint64, []byte) ([]uint64, error)
)

// CompressFloat 压缩 float64 数组（包装函数，自动检测约束）
func CompressFloat(dst []byte, src []float64) []byte {
	return compressFloatEntry(dst, src, CompressFloatWithConstraints)
}

// CompressFloatLZ4 提供与 CompressFloat 相同接口、以 LZ4 为后端
func CompressFloatLZ4(dst []byte, src []float64) []byte {
	return compressFloatEntry(dst, src, CompressFloatWithConstraintsLZ4)
}

// CompressFloatSnappy 提供与 CompressFloat 相同接口、以 Snappy 为后端
func CompressFloatSnappy(dst []byte, src []float64) []byte {
	return compressFloatEntry(dst, src, CompressFloatWithConstraintsSnappy)
}

// CompressFloatBrotli 提供与 CompressFloat 相同接口、以 Brotli 为后端
func CompressFloatBrotli(dst []byte, src []float64) []byte {
	return compressFloatEntry(dst, src, CompressFloatWithConstraintsBrotli)
}

// CompressFloatXZ 提供与 CompressFloat 相同接口、以 XZ 为后端
func CompressFloatXZ(dst []byte, src []float64) []byte {
	return compressFloatEntry(dst, src, CompressFloatWithConstraintsXZ)
}

// DecompressFloat 解压缩到 float64 数组（包装函数，从数据中恢复约束）
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	return decompressFloatEntry(dst, src, DecompressFloatWithConstraints)
}

// DecompressFloatLZ4 提供与 DecompressFloat 相同接口、以 LZ4 为后端
func DecompressFloatLZ4(dst []float64, src []byte) ([]float64, error) {
	return decompressFloatEntry(dst, src, DecompressFloatWithConstraintsLZ4)
}

// DecompressFloatSnappy 提供与 DecompressFloat 相同接口、以 Snappy 为后端
func DecompressFloatSnappy(dst []float64, src []byte) ([]float64, error) {
	return decompressFloatEntry(dst, src, DecompressFloatWithConstraintsSnappy)
}

// DecompressFloatBrotli 提供与 DecompressFloat 相同接口、以 Brotli 为后端
func DecompressFloatBrotli(dst []float64, src []byte) ([]float64, error) {
	return decompressFloatEntry(dst, src, DecompressFloatWithConstraintsBrotli)
}

// DecompressFloatXZ 提供与 DecompressFloat 相同接口、以 XZ 为后端
func DecompressFloatXZ(dst []float64, src []byte) ([]float64, error) {
	return decompressFloatEntry(dst, src, DecompressFloatWithConstraintsXZ)
}

// encodeConstraints 将约束信息编码为字节数组
func encodeConstraints(nc *NumericalConstraints) []byte {
	header := make([]byte, 32) // 基础 32 字节头部

	// 字节0: 约束标志位
	var flags byte
	if nc.HasConstraint(ConstraintPrecision) {
		flags |= 1 << 0
	}
	if nc.HasConstraint(ConstraintRange) {
		flags |= 1 << 1
	}
	if nc.HasConstraint(ConstraintMonotonicity) {
		flags |= 1 << 2
	}
	if nc.HasConstraint(ConstraintSign) {
		flags |= 1 << 3
	}
	if nc.HasConstraint(ConstraintDiscrete) {
		flags |= 1 << 4
	}
	if nc.HasConstraint(ConstraintEnumeration) {
		flags |= 1 << 5
	}
	if nc.HasConstraint(ConstraintSparse) {
		flags |= 1 << 6
	}
	header[0] = flags

	// 字节1: 精度值
	header[1] = byte(nc.Precision)

	// 字节2: 单调性
	header[2] = byte(nc.Monotonicity + 128) // 偏移128以支持负值

	// 字节3-10: MinValue
	binary.LittleEndian.PutUint64(header[3:11], math.Float64bits(nc.MinValue))

	// 字节11-18: MaxValue
	binary.LittleEndian.PutUint64(header[11:19], math.Float64bits(nc.MaxValue))

	// 字节19-26: DiscreteStep
	binary.LittleEndian.PutUint64(header[19:27], math.Float64bits(nc.DiscreteStep))

	// 字节28-31: 枚举值数量（若启用枚举约束）
	if nc.HasConstraint(ConstraintEnumeration) && len(nc.EnumerationValues) > 0 {
		binary.LittleEndian.PutUint32(header[28:32], uint32(len(nc.EnumerationValues)))
		enumBytes := make([]byte, len(nc.EnumerationValues)*8)
		for i, v := range nc.EnumerationValues {
			binary.LittleEndian.PutUint64(enumBytes[i*8:(i+1)*8], math.Float64bits(v))
		}
		header = append(header, enumBytes...)
	} else {
		binary.LittleEndian.PutUint32(header[28:32], 0)
	}

	// 字节27: 正负值标志
	var signFlags byte
	if nc.AllowPositive {
		signFlags |= 1
	}
	if nc.AllowNegative {
		signFlags |= 2
	}
	header[27] = signFlags

	return header
}

// decodeConstraints 从字节数组解码约束信息
func decodeConstraints(data []byte) (*NumericalConstraints, int) {
	if len(data) < 32 {
		return NewNumericalConstraints(), 0
	}

	nc := NewNumericalConstraints()

	// 解码标志位
	flags := data[0]
	nc.HasConstraints[ConstraintPrecision] = (flags & (1 << 0)) != 0
	nc.HasConstraints[ConstraintRange] = (flags & (1 << 1)) != 0
	nc.HasConstraints[ConstraintMonotonicity] = (flags & (1 << 2)) != 0
	nc.HasConstraints[ConstraintSign] = (flags & (1 << 3)) != 0
	nc.HasConstraints[ConstraintDiscrete] = (flags & (1 << 4)) != 0
	nc.HasConstraints[ConstraintEnumeration] = (flags & (1 << 5)) != 0
	nc.HasConstraints[ConstraintSparse] = (flags & (1 << 6)) != 0

	// 解码精度
	nc.Precision = int(data[1])

	// 解码单调性
	nc.Monotonicity = int(data[2]) - 128

	// 解码范围
	nc.MinValue = math.Float64frombits(binary.LittleEndian.Uint64(data[3:11]))
	nc.MaxValue = math.Float64frombits(binary.LittleEndian.Uint64(data[11:19]))

	// 解码离散步长
	nc.DiscreteStep = math.Float64frombits(binary.LittleEndian.Uint64(data[19:27]))

	// 解码正负值标志
	signFlags := data[27]
	nc.AllowPositive = (signFlags & 1) != 0
	nc.AllowNegative = (signFlags & 2) != 0

	offset := 32
	if nc.HasConstraint(ConstraintEnumeration) {
		enumCount := binary.LittleEndian.Uint32(data[28:32])
		required := offset + int(enumCount)*8
		if len(data) < required {
			enumCount = 0
		}
		if enumCount > 0 {
			values := make([]float64, enumCount)
			for i := 0; i < int(enumCount); i++ {
				start := offset + i*8
				bits := binary.LittleEndian.Uint64(data[start : start+8])
				values[i] = math.Float64frombits(bits)
			}
			valueCopy := make([]float64, len(values))
			copy(valueCopy, values)
			nc.SetEnumerationConstraint(valueCopy)
			offset = offset + int(enumCount)*8
		}
	}

	return nc, offset // 返回约束对象和头部大小
}

// numerical.go CompressFloat 压缩入口 定义数值约束
func compressFloatEntry(dst []byte, src []float64, handler compressWithConstraintsFunc) []byte {
	if len(src) == 0 {
		return dst
	}
	definedNC := NewNumericalConstraints()
	// 用户定义 数值约束
	definedNC.EnableConstraint(0)
	definedNC.EnableConstraint(3)
	definedNC.SetPrecisionConstraint(3)
	definedNC.SetMonotonicityConstraint(1)
	//
	if definedNC.IsConstraintValid() {
		fmt.Println("使用预定义的数值约束进行压缩")
		header := encodeConstraints(definedNC)
		compressed := handler(nil, src, definedNC)
		return append(header, compressed...)
	}
	fmt.Println("自动检测数值约束进行压缩")
	nc := DetectConstraints(src)
	header := encodeConstraints(nc)
	compressed := handler(nil, src, nc)
	return append(header, compressed...)
}

func decompressFloatEntry(dst []float64, src []byte, handler decompressWithConstraintsFunc) ([]float64, error) {
	if len(src) == 0 {
		return dst, nil
	}
	nc, headerSize := decodeConstraints(src)
	compressedData := src[headerSize:]
	return handler(dst, compressedData, nc)
}

func CompressFloatWithConstraints(dst []byte, src []float64, nc *NumericalConstraints) []byte {
	if len(src) == 0 {
		return dst
	}
	// 1. 预处理阶段
	// 根据约束进行预处理
	processed := preprocessData(src, nc)

	// 在 LZ77 前再做一次通用 Delta 编码以增强重复度
	// deltaEncoded := deltaEncodeForLZ77(processed)
	// 2. 变换阶段（旧版 LZ77 + Huffman 方案）
	// lz77Bytes := lz77.Compress(nil, deltaEncoded)
	// compressed := huffmanLib.CompressBytes(nil, lz77Bytes)

	// 现改用 zstd 直接处理字节流
	deltaBytes := uint64SliceToBytes(processed)
	compressed := gozstd.Compress(nil, deltaBytes)
	return compressed
}

func compressFloatWithConstraintsUint64Backend(dst []byte, src []float64, nc *NumericalConstraints, backend uint64BackendCompressor) []byte {
	if len(src) == 0 {
		return dst
	}
	processed := preprocessData(src, nc)
	return backend(dst[:0], processed)
}

// CompressFloatWithConstraintsLZ4 复用预处理流程，最终使用 lz4 作为后端
func CompressFloatWithConstraintsLZ4(dst []byte, src []float64, nc *NumericalConstraints) []byte {
	return compressFloatWithConstraintsUint64Backend(dst, src, nc, lz4codec.Compress)
}

// CompressFloatWithConstraintsSnappy 复用预处理流程，最终使用 snappy 作为后端
func CompressFloatWithConstraintsSnappy(dst []byte, src []float64, nc *NumericalConstraints) []byte {
	return compressFloatWithConstraintsUint64Backend(dst, src, nc, snappycodec.Compress)
}

// CompressFloatWithConstraintsBrotli 复用预处理流程，最终使用 brotli 作为后端
func CompressFloatWithConstraintsBrotli(dst []byte, src []float64, nc *NumericalConstraints) []byte {
	return compressFloatWithConstraintsUint64Backend(dst, src, nc, brotlicodec.Compress)
}

// CompressFloatWithConstraintsXZ 复用预处理流程，最终使用 xz 作为后端
func CompressFloatWithConstraintsXZ(dst []byte, src []float64, nc *NumericalConstraints) []byte {
	return compressFloatWithConstraintsUint64Backend(dst, src, nc, xzcodec.Compress)
}

// DecompressFloatWithConstraints 使用数值约束解压缩到 float64 数组
func DecompressFloatWithConstraints(dst []float64, src []byte, nc *NumericalConstraints) ([]float64, error) {
	if len(src) == 0 {
		return dst, nil
	}

	// 旧版：先通过熵编码→LZ77 反解
	// lz77Bytes, err := huffmanLib.DecompressBytes(nil, src)
	// processedDelta, err := lz77.Decompress(nil, lz77Bytes)

	// 现使用 zstd 直接解压字节流
	deltaBytes, err := gozstd.Decompress(nil, src)
	if err != nil {
		return dst, err
	}
	processedDelta, err := bytesToUint64Slice(deltaBytes)
	if err != nil {
		return dst, err
	}

	// 逆 Delta，恢复到预处理阶段的表示
	// processed := deltaDecodeForLZ77(processedDelta)

	// 根据约束进行后处理，恢复原始数据
	result := postprocessData(processedDelta, nc)

	dst = append(dst, result...)
	return dst, nil
}

func decompressFloatWithConstraintsUint64Backend(dst []float64, src []byte, nc *NumericalConstraints, backend uint64BackendDecompressor) ([]float64, error) {
	if len(src) == 0 {
		return dst, nil
	}
	processedDelta, err := backend(nil, src)
	if err != nil {
		return dst, err
	}
	result := postprocessData(processedDelta, nc)
	dst = append(dst, result...)
	return dst, nil
}

// DecompressFloatWithConstraintsLZ4 使用 lz4 后端解压
func DecompressFloatWithConstraintsLZ4(dst []float64, src []byte, nc *NumericalConstraints) ([]float64, error) {
	return decompressFloatWithConstraintsUint64Backend(dst, src, nc, lz4codec.Decompress)
}

// DecompressFloatWithConstraintsSnappy 使用 snappy 后端解压
func DecompressFloatWithConstraintsSnappy(dst []float64, src []byte, nc *NumericalConstraints) ([]float64, error) {
	return decompressFloatWithConstraintsUint64Backend(dst, src, nc, snappycodec.Decompress)
}

// DecompressFloatWithConstraintsBrotli 使用 brotli 后端解压
func DecompressFloatWithConstraintsBrotli(dst []float64, src []byte, nc *NumericalConstraints) ([]float64, error) {
	return decompressFloatWithConstraintsUint64Backend(dst, src, nc, brotlicodec.Decompress)
}

// DecompressFloatWithConstraintsXZ 使用 xz 后端解压
func DecompressFloatWithConstraintsXZ(dst []float64, src []byte, nc *NumericalConstraints) ([]float64, error) {
	return decompressFloatWithConstraintsUint64Backend(dst, src, nc, xzcodec.Decompress)
}

// preprocessData 根据约束预处理数据
func preprocessData(data []float64, nc *NumericalConstraints) []uint64 {
	result := make([]uint64, len(data))

	if nc.HasConstraint(ConstraintEnumeration) && len(nc.EnumerationValues) > 0 {
		fmt.Println("✓ 应用枚举值约束: 映射为枚举索引")
		valueToIndex := make(map[float64]uint64, len(nc.EnumerationValues))
		for idx, v := range nc.EnumerationValues {
			valueToIndex[v] = uint64(idx)
		}
		for i, v := range data {
			if idx, ok := valueToIndex[v]; ok {
				result[i] = idx
				continue
			}
			minDist := math.MaxFloat64
			bestIdx := uint64(0)
			for enumIdx, enumVal := range nc.EnumerationValues {
				dist := math.Abs(v - enumVal)
				if dist < minDist {
					minDist = dist
					bestIdx = uint64(enumIdx)
				}
			}
			result[i] = bestIdx
		}
		return result
	}

	// 打印调试信息：预处理
	// fmt.Println("\n=== 数据预处理（压缩） ===")
	// n := 10
	// if len(data) < n {
	// 	n = len(data)
	// }
	// fmt.Printf("原始数据前 %d 个值: ", n)
	// for i := 0; i < n; i++ {
	// 	fmt.Printf("%.6f ", data[i])
	// }
	// fmt.Println()

	// 如果启用了离散步长约束，优先使用离散步长转换
	if nc.HasConstraint(ConstraintDiscrete) && nc.DiscreteStep > 0 {
		fmt.Printf("✓ 应用离散步长约束: 步长 = %.6f, 基数 = %.6f\n", nc.DiscreteStep, nc.MinValue)
		// 找到基数（最小值）
		baseValue := nc.MinValue

		// 将每个值转换为: (值 - 基数) / 步长
		for i, v := range data {
			steps := (v - baseValue) / nc.DiscreteStep
			result[i] = uint64(int64(steps + 0.5)) // 四舍五入
		}

		// 打印转换后的值
		// fmt.Printf("离散步长转换后前 %d 个值: ", n)
		// for i := 0; i < n; i++ {
		// 	fmt.Printf("%d ", result[i])
		// }
		// fmt.Println()
	} else if nc.HasConstraint(ConstraintPrecision) && nc.Precision > 0 {
		fmt.Printf("✓ 应用精度约束: 小数点后 %d 位\n", nc.Precision)
		// 如果没有离散步长约束，但有精度约束，进行精度转换
		// 将浮点数转换为整数（乘以 10^precision）
		multiplier := 1.0
		for i := 0; i < nc.Precision; i++ {
			multiplier *= 10
		}
		fmt.Printf("乘数: %.0f\n", multiplier)

		for i, v := range data {
			// 转换为整数（四舍五入，避免浮点数精度误差）
			temp := v * multiplier
			if temp >= 0 {
				result[i] = uint64(int64(temp + 0.5))
			} else {
				result[i] = uint64(int64(temp - 0.5))
			}
		}

		// 打印转换后的值
		// fmt.Printf("精度转换后前 %d 个值: ", n)
		// for i := 0; i < n; i++ {
		// 	fmt.Printf("%d ", result[i])
		// }
		// fmt.Println()
	} else {
		fmt.Println("✓ 无约束，使用浮点数位表示")
		// 直接使用浮点数的位表示
		result = float64SliceToUint64(data)

		// 打印转换后的值
		// fmt.Printf("位表示前 %d 个值: ", n)
		// for i := 0; i < n; i++ {
		// 	fmt.Printf("%d ", result[i])
		// }
		// fmt.Println()
	}

	// 如果启用了单调性约束，转换为 delta 编码
	if nc.HasConstraint(ConstraintMonotonicity) && len(result) > 1 {
		fmt.Println("✓ 应用单调性约束: Delta 编码")
		deltas := make([]uint64, len(result))
		deltas[0] = result[0]

		for i := 1; i < len(result); i++ {
			// 计算差值
			if result[i] >= result[i-1] {
				deltas[i] = result[i] - result[i-1]
			} else {
				// 使用补码表示负差值
				deltas[i] = uint64(int64(result[i]) - int64(result[i-1]))
			}
		}

		// 打印 delta 编码后的值
		// fmt.Printf("Delta 编码后前 %d 个值: ", n)
		// for i := 0; i < n && i < len(deltas); i++ {
		// 	fmt.Printf("%d ", deltas[i])
		// }
		// fmt.Println()
		// fmt.Println("==================")
		// fmt.Println()

		return deltas
	}

	// fmt.Printf("最终预处理后前 %d 个值: ", n)
	// for i := 0; i < n; i++ {
	// 	fmt.Printf("%d ", result[i])
	// }
	// fmt.Println()
	// fmt.Println("==================")
	// fmt.Println()

	return result
}

// postprocessData 根据约束后处理数据，恢复原始值
func postprocessData(data []uint64, nc *NumericalConstraints) []float64 {
	result := make([]float64, len(data))

	if nc.HasConstraint(ConstraintEnumeration) && len(nc.EnumerationValues) > 0 {
		fmt.Println("✓ 恢复枚举值约束: 根据索引还原")
		for i, idx := range data {
			intIdx := int(idx)
			if intIdx >= 0 && intIdx < len(nc.EnumerationValues) {
				result[i] = nc.EnumerationValues[intIdx]
			} else if len(nc.EnumerationValues) > 0 {
				result[i] = nc.EnumerationValues[len(nc.EnumerationValues)-1]
			}
		}
		return result
	}

	// 打印调试信息：后处理
	// fmt.Println("\n=== 数据后处理（解压缩） ===")
	// n := 10
	// if len(data) < n {
	// 	n = len(data)
	// }
	// fmt.Printf("解压后的原始数据前 %d 个值: ", n)
	// for i := 0; i < n; i++ {
	// 	fmt.Printf("%d ", data[i])
	// }
	// fmt.Println()

	// 如果启用了单调性约束，先恢复 delta 编码
	processed := data
	if nc.HasConstraint(ConstraintMonotonicity) && len(data) > 1 {
		fmt.Println("✓ 恢复 Delta 编码")
		processed = make([]uint64, len(data))
		processed[0] = data[0]

		for i := 1; i < len(data); i++ {
			// 从差值恢复原值
			processed[i] = uint64(int64(processed[i-1]) + int64(data[i]))
		}

		// 打印恢复后的值
		// fmt.Printf("Delta 解码后前 %d 个值: ", n)
		// for i := 0; i < n; i++ {
		// 	fmt.Printf("%d ", processed[i])
		// }
		// fmt.Println()
	}

	// 如果启用了离散步长约束，优先使用离散步长恢复
	if nc.HasConstraint(ConstraintDiscrete) && nc.DiscreteStep > 0 {
		fmt.Printf("✓ 恢复离散步长约束: 步长 = %.6f, 基数 = %.6f\n", nc.DiscreteStep, nc.MinValue)

		for i, steps := range processed {
			result[i] = recoverDiscreteValue(nc.MinValue, nc.DiscreteStep, steps)
		}

		// 打印恢复后的浮点数
		// fmt.Printf("离散步长恢复后前 %d 个值: ", n)
		// for i := 0; i < n; i++ {
		// 	fmt.Printf("%.6f ", result[i])
		// }
		// fmt.Println()
	} else if nc.HasConstraint(ConstraintPrecision) && nc.Precision > 0 {
		// 如果没有离散步长约束，但有精度约束，进行精度恢复
		fmt.Printf("✓ 恢复精度约束: 小数点后 %d 位, 除数: %.0f\n", nc.Precision, math.Pow(10, float64(nc.Precision)))
		multiplier := 1.0
		for i := 0; i < nc.Precision; i++ {
			multiplier *= 10
		}

		for i, intVal := range processed {
			result[i] = float64(int64(intVal)) / multiplier
		}

		// 打印恢复后的浮点数
		// fmt.Printf("精度恢复后前 %d 个值: ", n)
		// for i := 0; i < n; i++ {
		// 	fmt.Printf("%.6f ", result[i])
		// }
		// fmt.Println()
	} else {
		fmt.Println("✓ 无约束，从位表示恢复浮点数")
		// 直接从位表示恢复浮点数
		result = uint64SliceToFloat64(processed)

		// 打印恢复后的浮点数
		// fmt.Printf("位恢复后前 %d 个值: ", n)
		// for i := 0; i < n; i++ {
		// 	fmt.Printf("%.6f ", result[i])
		// }
		// fmt.Println()
	}

	// fmt.Println("==================")
	// fmt.Println()

	return result
}

// 解压时根据离散步长推算最合理的小数位并四舍五入，彻底消除了 63.85500000000003 这类尾差
func recoverDiscreteValue(baseValue, step float64, steps uint64) float64 {
	if step == 0 {
		return baseValue
	}
	raw := baseValue + step*float64(steps)
	multiplier := discreteScaleMultiplier(step)
	if multiplier == 0 {
		return raw
	}
	return math.Round(raw*multiplier) / multiplier
}

// 解压时根据离散步长推算最合理的小数位并四舍五入，彻底消除了 63.85500000000003 这类尾差
func discreteScaleMultiplier(step float64) float64 {
	if step <= 0 {
		return 0
	}
	const maxDecimals = 15
	const tolerance = 1e-9
	for decimals := 0; decimals <= maxDecimals; decimals++ {
		multiplier := math.Pow(10, float64(decimals))
		scaled := step * multiplier
		if math.Abs(scaled-math.Round(scaled)) < tolerance {
			return multiplier
		}
	}
	return 0
}

func uint64SliceToBytes(data []uint64) []byte {
	if len(data) == 0 {
		return nil
	}
	buf := make([]byte, len(data)*8)
	for i, v := range data {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], v)
	}
	return buf
}

func bytesToUint64Slice(src []byte) ([]uint64, error) {
	if len(src)%8 != 0 {
		return nil, fmt.Errorf("numerical: zstd payload length %d not aligned to uint64", len(src))
	}
	count := len(src) / 8
	result := make([]uint64, count)
	for i := 0; i < count; i++ {
		result[i] = binary.LittleEndian.Uint64(src[i*8 : (i+1)*8])
	}
	return result, nil
}

// uint64SliceToFloat64 将 uint64 切片按位视为 float64
func uint64SliceToFloat64(src []uint64) []float64 {
	if len(src) == 0 {
		return nil
	}
	result := make([]float64, len(src))
	for i, v := range src {
		result[i] = math.Float64frombits(v)
	}
	return result
}

// float64SliceToUint64 执行与 uint64SliceToFloat64 相反的位还原
func float64SliceToUint64(src []float64) []uint64 {
	if len(src) == 0 {
		return nil
	}
	result := make([]uint64, len(src))
	for i, v := range src {
		result[i] = math.Float64bits(v)
	}
	return result
}
