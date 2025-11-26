package numerical

import (
	"encoding/binary"
	"fmt"
	"math"

	"myalgo/algorithms/ans"
)

// CompressFloat 压缩 float64 数组（包装函数，自动检测约束）
func CompressFloat(dst []byte, src []float64) []byte {
	if len(src) == 0 {
		return dst
	}

	// 自动检测数值约束
	nc := DetectConstraints(src)
	nc.PrintConstraints()
	// 将约束信息编码到压缩数据头部
	header := encodeConstraints(nc)

	// 调用带约束参数的压缩函数
	compressed := CompressFloatWithConstraints(nil, src, nc)

	// 打印大小信息
	fmt.Printf("\n=== 压缩大小分析 ===\n")
	fmt.Printf("控制信息大小: %d 字节 (约束头部)\n", len(header))
	fmt.Printf("有效压缩数据大小: %d 字节 (压缩后的数值数据)\n", len(compressed))
	fmt.Printf("总压缩大小: %d 字节\n", len(header)+len(compressed))
	fmt.Printf("控制信息占比: %.2f%%\n", float64(len(header))*100.0/float64(len(header)+len(compressed)))
	fmt.Println("==================")

	// 合并头部和压缩数据
	result := append(header, compressed...)
	return result
}

// DecompressFloat 解压缩到 float64 数组（包装函数，从数据中恢复约束）
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	if len(src) == 0 {
		return dst, nil
	}

	// 从压缩数据头部解码约束信息
	nc, headerSize := decodeConstraints(src)

	// 跳过头部，获取实际压缩数据
	compressedData := src[headerSize:]

	return DecompressFloatWithConstraints(dst, compressedData, nc)
}

// encodeConstraints 将约束信息编码为字节数组
func encodeConstraints(nc *NumericalConstraints) []byte {
	header := make([]byte, 32) // 预留32字节给约束信息

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

	return nc, 32 // 返回约束对象和头部大小
}

func CompressFloatWithConstraints(dst []byte, src []float64, nc *NumericalConstraints) []byte {
	if len(src) == 0 {
		return dst
	}

	// 根据约束进行预处理
	processed := preprocessData(src, nc)

	// 使用 ans (FSE) 编码压缩
	compressed := ans.Compress(nil, processed)

	return compressed
}

// DecompressFloatWithConstraints 使用数值约束解压缩到 float64 数组
func DecompressFloatWithConstraints(dst []float64, src []byte, nc *NumericalConstraints) ([]float64, error) {
	if len(src) == 0 {
		return dst, nil
	}

	// 使用 ans (FSE) 解码
	processed, err := ans.Decompress(nil, src)
	if err != nil {
		return dst, err
	}

	// 根据约束进行后处理，恢复原始数据
	result := postprocessData(processed, nc)

	dst = append(dst, result...)
	return dst, nil
}

// preprocessData 根据约束预处理数据
func preprocessData(data []float64, nc *NumericalConstraints) []uint64 {
	result := make([]uint64, len(data))

	// 打印调试信息：预处理
	fmt.Println("\n=== 数据预处理（压缩） ===")
	n := 10
	if len(data) < n {
		n = len(data)
	}
	fmt.Printf("原始数据前 %d 个值: ", n)
	for i := 0; i < n; i++ {
		fmt.Printf("%.6f ", data[i])
	}
	fmt.Println()

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
		fmt.Printf("离散步长转换后前 %d 个值: ", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%d ", result[i])
		}
		fmt.Println()
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
		fmt.Printf("精度转换后前 %d 个值: ", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%d ", result[i])
		}
		fmt.Println()
	} else {
		fmt.Println("✓ 无约束，使用浮点数位表示")
		// 直接使用浮点数的位表示
		for i, v := range data {
			result[i] = math.Float64bits(v)
		}

		// 打印转换后的值
		fmt.Printf("位表示前 %d 个值: ", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%d ", result[i])
		}
		fmt.Println()
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
		fmt.Printf("Delta 编码后前 %d 个值: ", n)
		for i := 0; i < n && i < len(deltas); i++ {
			fmt.Printf("%d ", deltas[i])
		}
		fmt.Println()
		fmt.Println("==================\n")

		return deltas
	}

	fmt.Printf("最终预处理后前 %d 个值: ", n)
	for i := 0; i < n; i++ {
		fmt.Printf("%d ", result[i])
	}
	fmt.Println()
	fmt.Println("==================\n")

	return result
}

// postprocessData 根据约束后处理数据，恢复原始值
func postprocessData(data []uint64, nc *NumericalConstraints) []float64 {
	result := make([]float64, len(data))

	// 打印调试信息：后处理
	fmt.Println("\n=== 数据后处理（解压缩） ===")
	n := 10
	if len(data) < n {
		n = len(data)
	}
	fmt.Printf("解压后的原始数据前 %d 个值: ", n)
	for i := 0; i < n; i++ {
		fmt.Printf("%d ", data[i])
	}
	fmt.Println()

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
		fmt.Printf("Delta 解码后前 %d 个值: ", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%d ", processed[i])
		}
		fmt.Println()
	}

	// 如果启用了离散步长约束，优先使用离散步长恢复
	if nc.HasConstraint(ConstraintDiscrete) && nc.DiscreteStep > 0 {
		fmt.Printf("✓ 恢复离散步长约束: 步长 = %.6f, 基数 = %.6f\n", nc.DiscreteStep, nc.MinValue)

		for i, steps := range processed {
			result[i] = recoverDiscreteValue(nc.MinValue, nc.DiscreteStep, steps)
		}

		// 打印恢复后的浮点数
		fmt.Printf("离散步长恢复后前 %d 个值: ", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%.6f ", result[i])
		}
		fmt.Println()
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
		fmt.Printf("精度恢复后前 %d 个值: ", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%.6f ", result[i])
		}
		fmt.Println()
	} else {
		fmt.Println("✓ 无约束，从位表示恢复浮点数")
		// 直接从位表示恢复浮点数
		for i, bits := range processed {
			result[i] = math.Float64frombits(bits)
		}

		// 打印恢复后的浮点数
		fmt.Printf("位恢复后前 %d 个值: ", n)
		for i := 0; i < n; i++ {
			fmt.Printf("%.6f ", result[i])
		}
		fmt.Println()
	}

	fmt.Println("==================\n")

	return result
}

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
