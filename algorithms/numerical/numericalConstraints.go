package numerical

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// NumericalConstraints 数值约束结构
// 根据图示包含：数据精度、数据范围、枚举值、单调性、正负值、离散值
type NumericalConstraints struct {
	// ========== 约束值 ==========

	// Precision 数据精度（小数点后位数 - 最大值）
	Precision int

	// PrecisionDistribution 精度分布统计 map[精度]出现次数
	PrecisionDistribution map[int]int

	// MinValue 数据范围 - 最小值
	MinValue float64

	// MaxValue 数据范围 - 最大值
	MaxValue float64

	// EnumerationValues 枚举值列表（离散的可能值）
	EnumerationValues []float64

	// Monotonicity 单调性
	// 0: 无单调性, 1: 单调递增, -1: 单调递减, 2: 严格递增, -2: 严格递减
	Monotonicity int

	// AllowPositive 是否允许正值
	AllowPositive bool

	// AllowNegative 是否允许负值
	AllowNegative bool

	// DiscreteStep 离散值步长（如果数据是离散的，如 0.5 的倍数）
	DiscreteStep float64

	// ========== 约束启用标志 ==========

	// HasConstraints 标记哪些约束被启用
	// 索引对应：0-精度, 1-范围, 2-枚举值, 3-单调性, 4-正负值, 5-离散值
	HasConstraints [6]bool
}

// 约束类型常量
const (
	ConstraintPrecision    = 0 // 数据精度
	ConstraintRange        = 1 // 数据范围
	ConstraintEnumeration  = 2 // 枚举值
	ConstraintMonotonicity = 3 // 单调性
	ConstraintSign         = 4 // 正负值
	ConstraintDiscrete     = 5 // 离散值
)

const (
	alpha       = 52 // double precision fractional bits
	maxDecimals = 16
)

var powerOf10Lookup = [18]uint64{
	1,
	10,
	100,
	1000,
	10000,
	100000,
	1000000,
	10000000,
	100000000,
	1000000000,
	10000000000,
	100000000000,
	1000000000000,
	10000000000000,
	100000000000000,
	1000000000000000,
	10000000000000000,
	100000000000000000,
}

// NewNumericalConstraints 创建默认的数值约束
func NewNumericalConstraints() *NumericalConstraints {
	return &NumericalConstraints{
		Precision:             -1, // -1 表示未指定
		PrecisionDistribution: make(map[int]int),
		MinValue:              0,
		MaxValue:              0,
		EnumerationValues:     nil,
		Monotonicity:          0, // 0 表示无单调性约束
		AllowPositive:         true,
		AllowNegative:         true,
		DiscreteStep:          0, // 0 表示连续值
		HasConstraints:        [6]bool{false, false, false, false, false, false},
	}
}

// EnableConstraint 启用指定的约束
func (nc *NumericalConstraints) EnableConstraint(constraintType int) {
	if constraintType >= 0 && constraintType < len(nc.HasConstraints) {
		nc.HasConstraints[constraintType] = true
	}
}

// DisableConstraint 禁用指定的约束
func (nc *NumericalConstraints) DisableConstraint(constraintType int) {
	if constraintType >= 0 && constraintType < len(nc.HasConstraints) {
		nc.HasConstraints[constraintType] = false
	}
}

// HasConstraint 检查是否启用了指定的约束
func (nc *NumericalConstraints) HasConstraint(constraintType int) bool {
	if constraintType >= 0 && constraintType < len(nc.HasConstraints) {
		return nc.HasConstraints[constraintType]
	}
	return false
}

// SetPrecisionConstraint 设置精度约束
func (nc *NumericalConstraints) SetPrecisionConstraint(precision int) {
	nc.Precision = precision
	nc.EnableConstraint(ConstraintPrecision)
}

// SetRangeConstraint 设置范围约束
func (nc *NumericalConstraints) SetRangeConstraint(min, max float64) {
	nc.MinValue = min
	nc.MaxValue = max
	nc.EnableConstraint(ConstraintRange)
}

// SetEnumerationConstraint 设置枚举值约束
func (nc *NumericalConstraints) SetEnumerationConstraint(values []float64) {
	nc.EnumerationValues = values
	nc.EnableConstraint(ConstraintEnumeration)
}

// SetMonotonicityConstraint 设置单调性约束
func (nc *NumericalConstraints) SetMonotonicityConstraint(monotonicity int) {
	nc.Monotonicity = monotonicity
	nc.EnableConstraint(ConstraintMonotonicity)
}

// SetSignConstraint 设置正负值约束
func (nc *NumericalConstraints) SetSignConstraint(allowPositive, allowNegative bool) {
	nc.AllowPositive = allowPositive
	nc.AllowNegative = allowNegative
	nc.EnableConstraint(ConstraintSign)
}

// SetDiscreteConstraint 设置离散值约束
func (nc *NumericalConstraints) SetDiscreteConstraint(step float64) {
	nc.DiscreteStep = step
	nc.EnableConstraint(ConstraintDiscrete)
}

// ========== 预处理和复原函数 ==========

// PreprocessPrecision 精度预处理：将浮点数转换为整数
// 精度为2（0.01）则每个数×100
func (nc *NumericalConstraints) PreprocessPrecision(data []float64) []int64 {
	if nc.Precision <= 0 {
		result := make([]int64, len(data))
		for i, v := range data {
			result[i] = int64(v)
		}
		return result
	}

	multiplier := 1.0
	for i := 0; i < nc.Precision; i++ {
		multiplier *= 10
	}

	result := make([]int64, len(data))
	for i, v := range data {
		result[i] = int64(v * multiplier)
	}
	return result
}

// RestorePrecision 精度复原：将整数转换回浮点数
func (nc *NumericalConstraints) RestorePrecision(data []int64) []float64 {
	if nc.Precision <= 0 {
		result := make([]float64, len(data))
		for i, v := range data {
			result[i] = float64(v)
		}
		return result
	}

	divisor := 1.0
	for i := 0; i < nc.Precision; i++ {
		divisor *= 10
	}

	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = float64(v) / divisor
	}
	return result
}

// PreprocessRange 范围预处理：每个数减去最小值
func (nc *NumericalConstraints) PreprocessRange(data []float64) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = v - nc.MinValue
	}
	return result
}

// RestoreRange 范围复原：每个数加上最小值
func (nc *NumericalConstraints) RestoreRange(data []float64) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = v + nc.MinValue
	}
	return result
}

// PreprocessEnumeration 枚举预处理：用索引代替浮点数
func (nc *NumericalConstraints) PreprocessEnumeration(data []float64) []int {
	result := make([]int, len(data))

	// 创建值到索引的映射
	valueToIndex := make(map[float64]int)
	for i, v := range nc.EnumerationValues {
		valueToIndex[v] = i
	}

	for i, v := range data {
		if idx, ok := valueToIndex[v]; ok {
			result[i] = idx
		} else {
			// 找最接近的枚举值
			minDist := 1e100
			minIdx := 0
			for j, enumVal := range nc.EnumerationValues {
				dist := v - enumVal
				if dist < 0 {
					dist = -dist
				}
				if dist < minDist {
					minDist = dist
					minIdx = j
				}
			}
			result[i] = minIdx
		}
	}
	return result
}

// RestoreEnumeration 枚举复原：用索引还原为浮点数
func (nc *NumericalConstraints) RestoreEnumeration(indices []int) []float64 {
	result := make([]float64, len(indices))
	for i, idx := range indices {
		if idx >= 0 && idx < len(nc.EnumerationValues) {
			result[i] = nc.EnumerationValues[idx]
		}
	}
	return result
}

// PreprocessMonotonicity 单调性预处理：转换为delta数组
// 单调递增：delta[i] = data[i+1] - data[i]
// 单调递减：delta[i] = -(data[i+1] - data[i]) = data[i] - data[i+1]
func (nc *NumericalConstraints) PreprocessMonotonicity(data []float64) ([]int64, float64) {
	if len(data) == 0 {
		return []int64{}, 0
	}

	baseValue := data[0]
	if len(data) == 1 {
		return []int64{}, baseValue
	}

	result := make([]int64, len(data)-1)

	if nc.Monotonicity < 0 {
		// 单调递减：delta = data[i] - data[i+1]
		for i := 0; i < len(data)-1; i++ {
			delta := data[i] - data[i+1]
			result[i] = int64(delta)
		}
	} else {
		// 单调递增：delta = data[i+1] - data[i]
		for i := 0; i < len(data)-1; i++ {
			delta := data[i+1] - data[i]
			result[i] = int64(delta)
		}
	}

	return result, baseValue
}

// RestoreMonotonicity 单调性复原：从delta数组还原原始数据
func (nc *NumericalConstraints) RestoreMonotonicity(deltas []int64, baseValue float64) []float64 {
	if len(deltas) == 0 {
		return []float64{baseValue}
	}

	result := make([]float64, len(deltas)+1)
	result[0] = baseValue

	if nc.Monotonicity < 0 {
		// 单调递减：data[i+1] = data[i] - delta[i]
		for i := 0; i < len(deltas); i++ {
			result[i+1] = result[i] - float64(deltas[i])
		}
	} else {
		// 单调递增：data[i+1] = data[i] + delta[i]
		for i := 0; i < len(deltas); i++ {
			result[i+1] = result[i] + float64(deltas[i])
		}
	}

	return result
}

// PreprocessDiscrete 离散值预处理：映射为整数（基数+步长倍数）
// 例如：步长0.5，基数0.2，值1.7 -> (1.7-0.2)/0.5 = 3
func (nc *NumericalConstraints) PreprocessDiscrete(data []float64) ([]int64, float64) {
	if nc.DiscreteStep == 0 || len(data) == 0 {
		result := make([]int64, len(data))
		for i, v := range data {
			result[i] = int64(v)
		}
		return result, 0
	}

	// 找到基数（数据中的最小值）
	baseValue := data[0]
	for _, v := range data {
		if v < baseValue {
			baseValue = v
		}
	}

	result := make([]int64, len(data))
	for i, v := range data {
		// (值 - 基数) / 步长 = 步数
		steps := (v - baseValue) / nc.DiscreteStep
		result[i] = int64(steps + 0.5) // 四舍五入
	}

	return result, baseValue
}

// RestoreDiscrete 离散值复原：从整数还原为浮点数
// 整数 * 步长 + 基数 = 原值
func (nc *NumericalConstraints) RestoreDiscrete(data []int64, baseValue float64) []float64 {
	result := make([]float64, len(data))
	for i, steps := range data {
		result[i] = baseValue + float64(steps)*nc.DiscreteStep
	}
	return result
}

// ========== 自动检测约束 ==========

// detectDecimalPlaces 检测一个浮点数的小数位数
func detectDecimalPlaces(value float64) int {
	if value < 0 {
		value = -value
	}

	// 如果是整数，返回 0
	if value == float64(int64(value)) {
		return 0
	}

	// 使用预计算的 10 的幂次表
	pow10 := [...]float64{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000, 10000000000, 100000000000, 1000000000000, 10000000000000, 100000000000000, 1000000000000000}

	// 从小数位数开始尝试（参考 ELF 的 getSignificantCount 方法）
	for i := 1; i < len(pow10); i++ {
		temp := value * pow10[i]
		tempLong := float64(int64(temp))

		// 如果乘以 10^i 后等于整数，说明至多有 i 位小数
		if tempLong == temp {
			// 双重验证：除回去应该等于原值（参考 ELF 的验证机制）
			if temp/pow10[i] != value {
				continue // 验证失败，继续尝试更高精度
			}

			// 去掉末尾的 0
			result := i
			for result > 0 && int64(tempLong)%10 == 0 {
				result--
				tempLong = float64(int64(tempLong) / 10)
			}
			return result
		}
	}

	return 15 // 超过精度范围
}

// detectDecimalPlacesFromString 从字符串直接检测小数位数（避免float64精度问题）
func detectDecimalPlacesFromString(str string) int {
	// 去掉前导空格和正负号
	str = strings.TrimSpace(str)
	if len(str) > 0 && (str[0] == '+' || str[0] == '-') {
		str = str[1:]
	}

	// 找到小数点
	dotIndex := strings.Index(str, ".")
	if dotIndex == -1 {
		return 0 // 没有小数点
	}

	// 计算小数点后的有效数字位数（去掉末尾的0）
	decimalPart := str[dotIndex+1:]
	decimalPart = strings.TrimRight(decimalPart, "0")

	return len(decimalPart)
}

// optimizedFractionToDecimal 基于 IEEE 754 二进制分数优化的小数位数检测
// 参考论文算法：通过二进制分数直接计算十进制小数位数
func optimizedFractionToDecimal(value float64) int {
	if value < 0 {
		value = -value
	}

	// 如果是整数，返回 0
	if value == float64(int64(value)) {
		return 0
	}
	bits := math.Float64bits(value)
	exp := int(bits>>52) - 1023
	if exp < 0 {
		exp = 0
	}
	fraction := bits << (12 + exp)
	fraction = fraction >> 4
	i := 1
	for i <= maxDecimals {
		fraction *= 0b1010
		decimal := int(fraction >> 60)
		fraction = fraction & 0x0fffffffffffffff
		bi := fraction >> (60 - alpha)
		if bi < powerOf10Lookup[i]/2 {
			return i
		}
		if ((1 << alpha) - bi) < powerOf10Lookup[i]/2 {
			if decimal == 9 {
				return i - 1
			} else {
				return i
			}
		}
		i++
	}
	return maxDecimals
}

// DetectConstraints 扫描浮点数组，自动检测并返回该数组的约束
func DetectConstraints(data []float64) *NumericalConstraints {
	if len(data) == 0 {
		return NewNumericalConstraints()
	}

	nc := NewNumericalConstraints()

	// 初始化检测变量
	minVal := data[0]
	maxVal := data[0]
	maxPrecision := 0
	hasPositive := false
	hasNegative := false

	// 单调性检测
	increasing := true
	decreasing := true
	strictIncreasing := true
	strictDecreasing := true

	// 枚举值检测
	valueSet := make(map[float64]bool)
	var uniqueValues []float64
	valueSet[data[0]] = true
	uniqueValues = append(uniqueValues, data[0])

	// 离散步长检测
	diffs := make(map[float64]int)
	tolerance := 1e-10

	// 一次循环完成所有检测
	for i := 0; i < len(data); i++ {
		v := data[i]

		// 1. 范围检测
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}

		// 2. 精度检测，并统计分布
		if v != float64(int64(v)) { // 不是整数
			// 使用新的基于 IEEE 754 二进制分数的精度检测方法
			precision := optimizedFractionToDecimal(v)
			nc.PrecisionDistribution[precision]++
			if precision > maxPrecision {
				maxPrecision = precision
			}
		} else {
			// 整数，精度为0
			nc.PrecisionDistribution[0]++
		}

		// 3. 正负值检测
		if v > 0 {
			hasPositive = true
		}
		if v < 0 {
			hasNegative = true
		}

		// 4. 枚举值检测
		if !valueSet[v] {
			valueSet[v] = true
			uniqueValues = append(uniqueValues, v)
		}

		// 5. 单调性和离散步长检测（需要前一个值）
		if i > 0 {
			prev := data[i-1]

			// 单调性
			if v < prev {
				increasing = false
				strictIncreasing = false
			}
			if v > prev {
				decreasing = false
				strictDecreasing = false
			}
			if v == prev {
				strictIncreasing = false
				strictDecreasing = false
			}

			// 离散步长
			diff := v - prev
			if diff < 0 {
				diff = -diff
			}
			if diff >= tolerance {
				found := false
				for existingDiff := range diffs {
					if abs(diff-existingDiff) < tolerance {
						diffs[existingDiff]++
						found = true
						break
					}
				}
				if !found {
					diffs[diff] = 1
				}
			}
		}
	}

	// 设置检测结果

	// 1. 精度 - 使用合理的精度，忽略异常高精度值
	if maxPrecision >= 0 {
		// 计算精度的合理值：
		// - 如果最大精度 >= 10，检查是否是异常值（出现次数很少）
		// - 使用 95% 分位数精度，而不是最大精度
		reasonablePrecision := maxPrecision

		if maxPrecision >= 10 {
			// 统计总数据量
			totalCount := 0
			for _, count := range nc.PrecisionDistribution {
				totalCount += count
			}

			// 从高精度往低精度找，直到累计数量超过 5% (允许 5% 的异常值)
			threshold := int(float64(totalCount) * 0.05)
			cumulativeCount := 0

			for precision := 15; precision >= 0; precision-- {
				if count, exists := nc.PrecisionDistribution[precision]; exists {
					cumulativeCount += count
					if cumulativeCount > threshold {
						// 找到了合理的精度上限
						reasonablePrecision = precision
						break
					}
				}
			}

			// 如果调整了精度，打印提示信息
			if reasonablePrecision != maxPrecision {
				fmt.Printf("[提示] 检测到 %d 位小数精度，但有 %d 个异常高精度值 (%.2f%%)，实际使用 %d 位精度\n",
					maxPrecision, cumulativeCount, float64(cumulativeCount)*100.0/float64(totalCount), reasonablePrecision)
			}
		}

		nc.SetPrecisionConstraint(reasonablePrecision)
	}

	// 2. 范围
	nc.SetRangeConstraint(minVal, maxVal)

	// 3. 枚举值（如果唯一值数量较少）
	if len(uniqueValues) > 0 && len(uniqueValues) <= len(data)/10 && len(uniqueValues) <= 100 {
		nc.SetEnumerationConstraint(uniqueValues)
	}

	// 4. 单调性
	var monotonicity int
	if strictIncreasing {
		monotonicity = 2
	} else if strictDecreasing {
		monotonicity = -2
	} else if increasing {
		monotonicity = 1
	} else if decreasing {
		monotonicity = -1
	}
	if monotonicity != 0 {
		nc.SetMonotonicityConstraint(monotonicity)
	}

	// 5. 正负值
	if !hasPositive || !hasNegative {
		nc.SetSignConstraint(hasPositive, hasNegative)
	}

	// 6. 离散步长
	if len(diffs) > 0 {
		// 找最小差值
		minDiff := 0.0
		for diff := range diffs {
			if minDiff == 0 || diff < minDiff {
				minDiff = diff
			}
		}

		// 检查是否所有差值都是minDiff的整数倍
		allMultiples := true
		for diff := range diffs {
			ratio := diff / minDiff
			if abs(ratio-float64(int64(ratio+0.5))) > tolerance {
				allMultiples = false
				break
			}
		}

		// 判断是否为离散步长
		// 增加条件：步长不能太小（相对于数据范围）
		// 如果步长 < 数据范围 / 10000，则不认为是有效的离散步长
		dataRange := maxVal - minVal
		isDiscrete := false

		if dataRange > 0 && minDiff > dataRange/10000 {
			// 只有当步长相对于数据范围合理时才考虑离散步长
			if allMultiples && len(diffs) > 1 {
				isDiscrete = true
			} else if len(diffs) <= 2 && len(diffs) < len(data)/5 {
				isDiscrete = true
			}
		}

		if isDiscrete {
			nc.SetDiscreteConstraint(minDiff)
		}
	}

	return nc
}

// PrintConstraints 打印约束信息
func (nc *NumericalConstraints) PrintConstraints() {
	fmt.Println("=== 数值约束信息 ===")

	// 打印精度约束
	if nc.HasConstraint(ConstraintPrecision) {
		fmt.Printf("✓ 数据精度: 小数点后最多 %d 位\n", nc.Precision)

		if len(nc.PrecisionDistribution) > 0 {
			fmt.Println("  精度分布:")
			printPrecisionDistribution(nc.PrecisionDistribution)
		}
	} else {
		fmt.Println("✗ 数据精度: 未启用")
	}

	// 打印范围约束
	if nc.HasConstraint(ConstraintRange) {
		fmt.Printf("✓ 数据范围: [%.6f, %.6f]\n", nc.MinValue, nc.MaxValue)
	} else {
		fmt.Println("✗ 数据范围: 未启用")
	}

	// 打印枚举值约束
	if nc.HasConstraint(ConstraintEnumeration) {
		fmt.Print("✓ 枚举值: [")
		for i, v := range nc.EnumerationValues {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%.6f", v)
		}
		fmt.Println("]")
	} else {
		fmt.Println("✗ 枚举值: 未启用")
	}

	// 打印单调性约束
	if nc.HasConstraint(ConstraintMonotonicity) {
		monotonicityStr := ""
		switch nc.Monotonicity {
		case 1:
			monotonicityStr = "单调递增"
		case -1:
			monotonicityStr = "单调递减"
		case 2:
			monotonicityStr = "严格递增"
		case -2:
			monotonicityStr = "严格递减"
		default:
			monotonicityStr = "无单调性"
		}
		fmt.Printf("✓ 单调性: %s\n", monotonicityStr)
	} else {
		fmt.Println("✗ 单调性: 未启用")
	}

	// 打印正负值约束
	if nc.HasConstraint(ConstraintSign) {
		signStr := ""
		if nc.AllowPositive && nc.AllowNegative {
			signStr = "正负值均可"
		} else if nc.AllowPositive {
			signStr = "仅正值"
		} else if nc.AllowNegative {
			signStr = "仅负值"
		} else {
			signStr = "仅零值"
		}
		fmt.Printf("✓ 正负值: %s\n", signStr)
	} else {
		fmt.Println("✗ 正负值: 未启用")
	}

	// 打印离散值约束
	if nc.HasConstraint(ConstraintDiscrete) {
		fmt.Printf("✓ 离散值: 步长 %.6f\n", nc.DiscreteStep)
	} else {
		fmt.Println("✗ 离散值: 未启用")
	}

	fmt.Println("==================")
}

func printPrecisionDistribution(dist map[int]int) {
	if len(dist) == 0 {
		return
	}

	keys := make([]int, 0, len(dist))
	for precision := range dist {
		keys = append(keys, precision)
	}
	sort.Ints(keys)

	total := 0
	for _, count := range dist {
		total += count
	}
	if total == 0 {
		return
	}

	for _, precision := range keys {
		count := dist[precision]
		percentage := float64(count) * 100.0 / float64(total)
		fmt.Printf("     %d 位: %d 个 (%.2f%%)\n", precision, count, percentage)
	}
}

// abs 返回浮点数的绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// DetectConstraintsWithStrings 从浮点数组和对应的字符串数组检测约束
// strings 参数用于精确检测小数位数（避免float64精度问题）
func DetectConstraintsWithStrings(data []float64, dataStrings []string) *NumericalConstraints {
	if len(data) == 0 {
		return NewNumericalConstraints()
	}

	nc := NewNumericalConstraints()

	// 初始化检测变量
	minVal := data[0]
	maxVal := data[0]
	maxPrecision := 0
	hasPositive := false
	hasNegative := false

	// 单调性检测
	increasing := true
	decreasing := true
	strictIncreasing := true
	strictDecreasing := true

	// 枚举值检测
	valueSet := make(map[float64]bool)
	var uniqueValues []float64
	valueSet[data[0]] = true
	uniqueValues = append(uniqueValues, data[0])

	// 离散步长检测
	diffs := make(map[float64]int)
	tolerance := 1e-10

	// 一次循环完成所有检测
	for i := 0; i < len(data); i++ {
		v := data[i]

		// 1. 范围检测
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}

		// 2. 精度检测 - 使用字符串方法，并统计分布
		if i < len(dataStrings) {
			precision := detectDecimalPlacesFromString(dataStrings[i])
			nc.PrecisionDistribution[precision]++
			if precision > maxPrecision {
				maxPrecision = precision
			}
		}

		// 3. 正负值检测
		if v > 0 {
			hasPositive = true
		}
		if v < 0 {
			hasNegative = true
		}

		// 4. 枚举值检测
		if !valueSet[v] {
			valueSet[v] = true
			uniqueValues = append(uniqueValues, v)
		}

		// 5. 单调性和离散步长检测（需要前一个值）
		if i > 0 {
			prev := data[i-1]

			// 单调性
			if v < prev {
				increasing = false
				strictIncreasing = false
			}
			if v > prev {
				decreasing = false
				strictDecreasing = false
			}
			if v == prev {
				strictIncreasing = false
				strictDecreasing = false
			}

			// 离散步长
			diff := v - prev
			if diff < 0 {
				diff = -diff
			}
			if diff >= tolerance {
				found := false
				for existingDiff := range diffs {
					if abs(diff-existingDiff) < tolerance {
						diffs[existingDiff]++
						found = true
						break
					}
				}
				if !found {
					diffs[diff] = 1
				}
			}
		}
	}

	// 设置检测结果

	// 1. 精度 - 使用合理的精度，忽略异常高精度值
	reasonablePrecision := maxPrecision
	if maxPrecision >= 10 {
		// 统计总数据量
		totalCount := 0
		for _, count := range nc.PrecisionDistribution {
			totalCount += count
		}

		// 从高精度往低精度找，直到累计数量超过 5% (允许 5% 的异常值)
		threshold := int(float64(totalCount) * 0.05)
		cumulativeCount := 0

		for precision := 15; precision >= 0; precision-- {
			if count, exists := nc.PrecisionDistribution[precision]; exists {
				cumulativeCount += count
				if cumulativeCount > threshold {
					// 找到了合理的精度上限
					reasonablePrecision = precision
					break
				}
			}
		}

		// 如果调整了精度，打印提示信息
		if reasonablePrecision != maxPrecision {
			fmt.Printf("[提示] 检测到 %d 位小数精度，但有 %d 个异常高精度值 (%.2f%%)，实际使用 %d 位精度\n",
				maxPrecision, cumulativeCount, float64(cumulativeCount)*100.0/float64(totalCount), reasonablePrecision)
		}
	}

	nc.SetPrecisionConstraint(reasonablePrecision)
	nc.SetRangeConstraint(minVal, maxVal)

	// 设置正负值约束
	nc.AllowPositive = hasPositive
	nc.AllowNegative = hasNegative
	if hasPositive || hasNegative {
		nc.EnableConstraint(ConstraintSign)
	}

	// 检测单调性约束
	if strictIncreasing {
		nc.SetMonotonicityConstraint(2) // 严格递增
	} else if strictDecreasing {
		nc.SetMonotonicityConstraint(-2) // 严格递减
	} else if increasing {
		nc.SetMonotonicityConstraint(1) // 单调递增
	} else if decreasing {
		nc.SetMonotonicityConstraint(-1) // 单调递减
	}

	// 检测枚举值约束（如果唯一值数量很少）
	if len(uniqueValues) <= 10 && len(uniqueValues) < len(data)/10 {
		nc.SetEnumerationConstraint(uniqueValues)
	}

	// 检测离散步长约束
	if len(diffs) > 0 {
		// 找出现最多的步长
		maxCount := 0
		var mostCommonStep float64
		for step, count := range diffs {
			if count > maxCount {
				maxCount = count
				mostCommonStep = step
			}
		}

		// 如果这个步长出现次数超过总步数的50%，认为是离散步长
		if maxCount > len(data)/2 {
			nc.SetDiscreteConstraint(mostCommonStep)
		}
	}

	return nc
}

// AnomalyInfo 异常值信息
type AnomalyInfo struct {
	Index          int     // 异常值在数组中的索引
	Value          float64 // 异常值
	Reason         string  // 异常原因描述
	ConstraintType int     // 违反的约束类型
}

// ValidateConstraints 使用约束检查数据，返回异常值信息
// 返回：异常值个数、异常值数组
func (nc *NumericalConstraints) ValidateConstraints(data []float64, dataStrings []string) (int, []AnomalyInfo) {
	var anomalies []AnomalyInfo

	for i, v := range data {
		// 1. 检查范围约束
		if nc.HasConstraint(ConstraintRange) {
			if v < nc.MinValue || v > nc.MaxValue {
				anomalies = append(anomalies, AnomalyInfo{
					Index:          i,
					Value:          v,
					Reason:         fmt.Sprintf("超出范围 [%.6f, %.6f]", nc.MinValue, nc.MaxValue),
					ConstraintType: ConstraintRange,
				})
				continue // 已经是异常值，跳过其他检查
			}
		}

		// 2. 检查精度约束
		if nc.HasConstraint(ConstraintPrecision) && i < len(dataStrings) {
			precision := detectDecimalPlacesFromString(dataStrings[i])
			if precision > nc.Precision {
				anomalies = append(anomalies, AnomalyInfo{
					Index:          i,
					Value:          v,
					Reason:         fmt.Sprintf("精度超出限制 (实际: %d 位, 限制: %d 位)", precision, nc.Precision),
					ConstraintType: ConstraintPrecision,
				})
				continue
			}
		}

		// 3. 检查枚举值约束
		if nc.HasConstraint(ConstraintEnumeration) {
			found := false
			for _, enumVal := range nc.EnumerationValues {
				if v == enumVal {
					found = true
					break
				}
			}
			if !found {
				anomalies = append(anomalies, AnomalyInfo{
					Index:          i,
					Value:          v,
					Reason:         "不在枚举值列表中",
					ConstraintType: ConstraintEnumeration,
				})
				continue
			}
		}

		// 4. 检查单调性约束（需要前一个值）
		if nc.HasConstraint(ConstraintMonotonicity) && i > 0 {
			prev := data[i-1]
			violated := false
			reason := ""

			switch nc.Monotonicity {
			case 1: // 单调递增
				if v < prev {
					violated = true
					reason = fmt.Sprintf("违反单调递增约束 (前值: %.6f, 当前值: %.6f)", prev, v)
				}
			case -1: // 单调递减
				if v > prev {
					violated = true
					reason = fmt.Sprintf("违反单调递减约束 (前值: %.6f, 当前值: %.6f)", prev, v)
				}
			case 2: // 严格递增
				if v <= prev {
					violated = true
					reason = fmt.Sprintf("违反严格递增约束 (前值: %.6f, 当前值: %.6f)", prev, v)
				}
			case -2: // 严格递减
				if v >= prev {
					violated = true
					reason = fmt.Sprintf("违反严格递减约束 (前值: %.6f, 当前值: %.6f)", prev, v)
				}
			}

			if violated {
				anomalies = append(anomalies, AnomalyInfo{
					Index:          i,
					Value:          v,
					Reason:         reason,
					ConstraintType: ConstraintMonotonicity,
				})
				continue
			}
		}

		// 5. 检查正负值约束
		if nc.HasConstraint(ConstraintSign) {
			if v > 0 && !nc.AllowPositive {
				anomalies = append(anomalies, AnomalyInfo{
					Index:          i,
					Value:          v,
					Reason:         "不允许正值",
					ConstraintType: ConstraintSign,
				})
				continue
			}
			if v < 0 && !nc.AllowNegative {
				anomalies = append(anomalies, AnomalyInfo{
					Index:          i,
					Value:          v,
					Reason:         "不允许负值",
					ConstraintType: ConstraintSign,
				})
				continue
			}
		}

		// 6. 检查离散值约束
		if nc.HasConstraint(ConstraintDiscrete) && i > 0 {
			// 找到最小值作为基准
			baseValue := data[0]
			for _, val := range data {
				if val < baseValue {
					baseValue = val
				}
			}

			// 检查是否是离散步长的倍数
			diff := v - baseValue
			if diff < 0 {
				diff = -diff
			}

			// 计算应该的步数
			steps := diff / nc.DiscreteStep
			expectedSteps := float64(int64(steps + 0.5))

			// 如果不是整数倍，则违反约束
			if abs(steps-expectedSteps) > 1e-9 {
				anomalies = append(anomalies, AnomalyInfo{
					Index:          i,
					Value:          v,
					Reason:         fmt.Sprintf("不符合离散步长 %.6f", nc.DiscreteStep),
					ConstraintType: ConstraintDiscrete,
				})
				continue
			}
		}
	}

	return len(anomalies), anomalies
}

// PrintAnomalies 打印异常值信息
func PrintAnomalies(count int, anomalies []AnomalyInfo) {
	fmt.Printf("\n=== 异常值检测结果 ===\n")
	fmt.Printf("异常值个数: %d\n", count)

	if count > 0 {
		fmt.Println("\n异常值详情:")
		for _, anomaly := range anomalies {
			constraintName := ""
			switch anomaly.ConstraintType {
			case ConstraintPrecision:
				constraintName = "精度约束"
			case ConstraintRange:
				constraintName = "范围约束"
			case ConstraintEnumeration:
				constraintName = "枚举值约束"
			case ConstraintMonotonicity:
				constraintName = "单调性约束"
			case ConstraintSign:
				constraintName = "正负值约束"
			case ConstraintDiscrete:
				constraintName = "离散值约束"
			}
			fmt.Printf("  [%d] 值: %.6f | 违反: %s | 原因: %s\n",
				anomaly.Index, anomaly.Value, constraintName, anomaly.Reason)
		}
	} else {
		fmt.Println("所有数据均符合约束 ✓")
	}

	fmt.Println("========================")
}
