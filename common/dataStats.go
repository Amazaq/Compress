package common

import (
	"math"
	"sort"
)

// TimeSeriesStats 时序数据统计信息
type TimeSeriesStats struct {
	// 基本统计信息
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Mean     float64 `json:"mean"`
	Median   float64 `json:"median"`
	StdDev   float64 `json:"std_dev"`
	Variance float64 `json:"variance"`
	Skewness float64 `json:"skewness"` // 偏度
	Kurtosis float64 `json:"kurtosis"` // 峰度
	Range    float64 `json:"range"`    // Max - Min
	IQR      float64 `json:"iqr"`      // 四分位距
	Q1       float64 `json:"q1"`       // 第一四分位数
	Q3       float64 `json:"q3"`       // 第三四分位数
	// 数值特性
	UniqueCount  int     `json:"unique_count"`  // 唯一值数量
	UniqueRatio  float64 `json:"unique_ratio"`  // 唯一值比率
	ZeroCount    int     `json:"zero_count"`    // 零值数量
	ZeroRatio    float64 `json:"zero_ratio"`    // 零值比率
	IntegerCount int     `json:"integer_count"` // 整数值数量
	IntegerRatio float64 `json:"integer_ratio"` // 整数值比率
	// 差分统计
	DiffStats       *DifferenceStats `json:"diff_stats"`
	SecondDiffStats *DifferenceStats `json:"second_diff_stats"`
	// 时序特征
	Monotonicity float64 `json:"monotonicity"`  // 单调性 [-1, 1]
	Smoothness   float64 `json:"smoothness"`    // 平滑度（二阶差分标准差）
	ChangePoints int     `json:"change_points"` // 变化点数量
	// 游程统计
	RunLength *RunLengthStats `json:"run_length"`
	// 位级统计
	BitStats *BitLevelStats `json:"bit_stats"`
	// 分布特征
	Entropy       float64 `json:"entropy"` // 信息熵
	PercentileP95 float64 `json:"percentile_95"`
	PercentileP5  float64 `json:"percentile_5"`
	// 相关性
	AutoCorrelation []float64 `json:"auto_correlation"` // 自相关系数(lag 1-10)
	Periodicity     int       `json:"periodicity"`      // 主周期长度
	PeriodicScore   float64   `json:"periodic_score"`   // 周期性强度
}

// DifferenceStats 差分统计信息
type DifferenceStats struct {
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Mean        float64 `json:"mean"`
	StdDev      float64 `json:"std_dev"`
	Range       float64 `json:"range"`
	ZeroRatio   float64 `json:"zero_ratio"`   // 差分为0的比率
	UniqueCount int     `json:"unique_count"` // 唯一值数量
	UniqueRatio float64 `json:"unique_ratio"` // 唯一值比率
}

// RunLengthStats 游程统计信息
type RunLengthStats struct {
	MaxRunLength     int     `json:"max_run_length"`     // 最长游程
	AvgRunLength     float64 `json:"avg_run_length"`     // 平均游程长度
	RunCount         int     `json:"run_count"`          // 游程数量
	ConstantRunRatio float64 `json:"constant_run_ratio"` // 常数游程比率
	InRunRatio       float64
}

// BitLevelStats 位级统计信息
type BitLevelStats struct {
	AvgSetBits      float64 `json:"avg_set_bits"`     // 平均设置位数
	SignChanges     int     `json:"sign_changes"`     // 符号变化次数
	MantissaEntropy float64 `json:"mantissa_entropy"` // 尾数部分熵
	ExponentRange   int     `json:"exponent_range"`   // 指数范围
	CommonExponent  int     `json:"common_exponent"`  // 最常见指数
}

// AnalyzeTimeSeries 分析时序数据的统计特征
// 优化版本：尽可能在最少的扫描次数中完成所有计算
func AnalyzeTimeSeries(src []float64) *TimeSeriesStats {
	if len(src) == 0 {
		return &TimeSeriesStats{}
	}

	n := len(src)
	stats := &TimeSeriesStats{
		Min: math.Inf(1),
		Max: math.Inf(-1),
	}

	// 初始化差分统计
	stats.DiffStats = &DifferenceStats{
		Min: math.Inf(1),
		Max: math.Inf(-1),
	}
	stats.SecondDiffStats = &DifferenceStats{
		Min: math.Inf(1),
		Max: math.Inf(-1),
	}

	// 初始化游程统计
	stats.RunLength = &RunLengthStats{}

	// 初始化位级统计
	stats.BitStats = &BitLevelStats{}

	// === 第一遍扫描：收集所有可以在单次遍历中计算的统计量 ===
	var sum float64
	uniqueMap := make(map[float64]bool)
	validCount := 0

	// 差分相关
	diff1Map := make(map[float64]bool)
	diff2Map := make(map[float64]bool)
	var diff1Sum, diff2Sum float64
	diff1ZeroCount := 0
	diff2ZeroCount := 0
	var prevValue, prevDiff1 float64

	// 时序特征
	increasing := 0
	decreasing := 0
	changePoints := 0
	prevTrend := 0

	// 游程统计
	currentRun := 1
	totalRun := 0
	constantRuns := 0
	singleCount := 0

	// 位级统计
	signChanges := 0
	totalSetBits := 0
	exponentMap := make(map[int]int)
	mantissaMap := make(map[uint64]int)

	// 熵计算用的直方图
	bins := 100
	hist := make([]int, bins)
	var histMin, histMax float64 = math.Inf(1), math.Inf(-1)

	for i, v := range src {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}

		// === 基本统计 ===
		if v < stats.Min {
			stats.Min = v
		}
		if v > stats.Max {
			stats.Max = v
		}
		sum += v
		validCount++

		// === 值属性统计 ===
		uniqueMap[v] = true
		if v == 0 {
			stats.ZeroCount++
		}
		if v == math.Floor(v) {
			stats.IntegerCount++
		}

		// === 直方图（用于熵计算）===
		if v < histMin {
			histMin = v
		}
		if v > histMax {
			histMax = v
		}

		// === 一阶差分统计 ===
		if i > 0 {
			diff1 := v - prevValue
			diff1Map[diff1] = true

			if diff1 < stats.DiffStats.Min {
				stats.DiffStats.Min = diff1
			}
			if diff1 > stats.DiffStats.Max {
				stats.DiffStats.Max = diff1
			}
			diff1Sum += diff1

			if math.Abs(diff1) < 1e-10 {
				diff1ZeroCount++
			}

			// === 时序特征（单调性和变化点）===
			currentTrend := 0
			if diff1 > 0 {
				increasing++
				currentTrend = 1
			} else if diff1 < 0 {
				decreasing++
				currentTrend = -1
			}

			if prevTrend != 0 && currentTrend != 0 && prevTrend != currentTrend {
				changePoints++
			}
			if currentTrend != 0 {
				prevTrend = currentTrend
			}

			// === 游程统计 ===
			if v == prevValue {
				currentRun++
			} else {
				if currentRun == 1 {
					singleCount++
				} else {
					constantRuns++
				}
				if currentRun > stats.RunLength.MaxRunLength {
					stats.RunLength.MaxRunLength = currentRun
				}
				totalRun += currentRun
				stats.RunLength.RunCount++
				currentRun = 1
			}

			// === 二阶差分统计 ===
			if i > 1 {
				diff2 := diff1 - prevDiff1
				diff2Map[diff2] = true

				if diff2 < stats.SecondDiffStats.Min {
					stats.SecondDiffStats.Min = diff2
				}
				if diff2 > stats.SecondDiffStats.Max {
					stats.SecondDiffStats.Max = diff2
				}
				diff2Sum += diff2

				if math.Abs(diff2) < 1e-10 {
					diff2ZeroCount++
				}
			}

			prevDiff1 = diff1
		}

		// === 位级统计 ===
		if i > 0 && ((v < 0 && prevValue >= 0) || (v >= 0 && prevValue < 0)) {
			signChanges++
		}

		bits := math.Float64bits(v)
		totalSetBits += popcount(bits)

		exponent := int((bits>>52)&0x7FF) - 1023
		exponentMap[exponent]++

		mantissa := bits & 0xFFFFFFFFFFFFF
		mantissaMap[mantissa]++

		prevValue = v
	}

	// === 处理最后一个游程 ===
	if currentRun > 1 {
		constantRuns++
	}
	if currentRun > stats.RunLength.MaxRunLength {
		stats.RunLength.MaxRunLength = currentRun
	}
	totalRun += currentRun
	stats.RunLength.RunCount++

	// === 完成第一遍扫描后的计算 ===
	if validCount == 0 {
		return stats
	}

	// 基本统计
	stats.Mean = sum / float64(validCount)
	stats.Range = stats.Max - stats.Min

	// 值属性
	stats.UniqueCount = len(uniqueMap)
	stats.UniqueRatio = float64(stats.UniqueCount) / float64(n)
	stats.ZeroRatio = float64(stats.ZeroCount) / float64(n)
	stats.IntegerRatio = float64(stats.IntegerCount) / float64(n)

	// 一阶差分
	diff1Count := n - 1
	if diff1Count > 0 {
		stats.DiffStats.UniqueCount = len(diff1Map)
		stats.DiffStats.UniqueRatio = float64(stats.DiffStats.UniqueCount) / float64(n)
		stats.DiffStats.Mean = diff1Sum / float64(diff1Count)
		stats.DiffStats.Range = stats.DiffStats.Max - stats.DiffStats.Min
		stats.DiffStats.ZeroRatio = float64(diff1ZeroCount) / float64(diff1Count)
	}

	// 二阶差分
	diff2Count := n - 2
	if diff2Count > 0 {
		stats.SecondDiffStats.UniqueCount = len(diff2Map)
		stats.SecondDiffStats.UniqueRatio = float64(stats.SecondDiffStats.UniqueCount) / float64(n)
		stats.SecondDiffStats.Mean = diff2Sum / float64(diff2Count)
		stats.SecondDiffStats.Range = stats.SecondDiffStats.Max - stats.SecondDiffStats.Min
		stats.SecondDiffStats.ZeroRatio = float64(diff2ZeroCount) / float64(diff2Count)
	}

	// 时序特征
	total := increasing + decreasing
	if total > 0 {
		stats.Monotonicity = float64(increasing-decreasing) / float64(total)
	}
	stats.ChangePoints = changePoints

	// 游程统计
	if stats.RunLength.RunCount > 0 {
		stats.RunLength.AvgRunLength = float64(totalRun) / float64(stats.RunLength.RunCount)
		stats.RunLength.ConstantRunRatio = float64(constantRuns) / float64(stats.RunLength.RunCount)
		stats.RunLength.InRunRatio = 1.0 - float64(singleCount)/float64(n)
	}

	// 位级统计
	stats.BitStats.SignChanges = signChanges
	stats.BitStats.AvgSetBits = float64(totalSetBits) / float64(validCount)

	minExp, maxExp := math.MaxInt32, math.MinInt32
	maxCount := 0
	for exp, count := range exponentMap {
		if exp < minExp {
			minExp = exp
		}
		if exp > maxExp {
			maxExp = exp
		}
		if count > maxCount {
			maxCount = count
			stats.BitStats.CommonExponent = exp
		}
	}
	stats.BitStats.ExponentRange = maxExp - minExp
	stats.BitStats.MantissaEntropy = calculateMapEntropy(mantissaMap, validCount)

	// 熵计算 - 填充直方图
	if histMin != histMax && validCount > 0 {
		binWidth := (histMax - histMin) / float64(bins)
		for _, v := range src {
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				binIdx := int((v - histMin) / binWidth)
				if binIdx >= bins {
					binIdx = bins - 1
				}
				hist[binIdx]++
			}
		}

		entropy := 0.0
		for _, count := range hist {
			if count > 0 {
				p := float64(count) / float64(validCount)
				entropy -= p * math.Log2(p)
			}
		}
		stats.Entropy = entropy
	}

	// === 第二遍扫描：需要均值的统计量（方差、标准差、偏度、峰度）===
	var sumSqDiff, sumCubeDiff, sumQuadDiff float64
	var diff1SumSqDiff, diff2SumSqDiff float64

	prevValue = 0
	prevDiff1 = 0
	for i, v := range src {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}

		// 基本统计的方差等
		diff := v - stats.Mean
		sumSqDiff += diff * diff
		sumCubeDiff += diff * diff * diff
		sumQuadDiff += diff * diff * diff * diff

		// 一阶差分的标准差
		if i > 0 {
			diff1 := v - prevValue
			d := diff1 - stats.DiffStats.Mean
			diff1SumSqDiff += d * d

			// 二阶差分的标准差
			if i > 1 {
				diff2 := diff1 - prevDiff1
				d2 := diff2 - stats.SecondDiffStats.Mean
				diff2SumSqDiff += d2 * d2
			}
			prevDiff1 = diff1
		}

		prevValue = v
	}

	// 完成方差相关计算
	if validCount > 1 {
		stats.Variance = sumSqDiff / float64(validCount)
		stats.StdDev = math.Sqrt(stats.Variance)

		if stats.StdDev > 0 {
			stats.Skewness = (sumCubeDiff / float64(validCount)) / math.Pow(stats.StdDev, 3)
			stats.Kurtosis = (sumQuadDiff / float64(validCount)) / math.Pow(stats.StdDev, 4)
		}
	}

	if diff1Count > 0 {
		stats.DiffStats.StdDev = math.Sqrt(diff1SumSqDiff / float64(diff1Count))
	}

	if diff2Count > 0 {
		stats.SecondDiffStats.StdDev = math.Sqrt(diff2SumSqDiff / float64(diff2Count))
		// 平滑度
		stats.Smoothness = stats.SecondDiffStats.StdDev
	}

	// === 第三遍扫描：需要排序的统计量（中位数、四分位数）===
	sorted := make([]float64, 0, validCount)
	for _, v := range src {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			sorted = append(sorted, v)
		}
	}
	sort.Float64s(sorted)

	if len(sorted) > 0 {
		stats.Median = percentile(sorted, 50)
		stats.Q1 = percentile(sorted, 25)
		stats.Q3 = percentile(sorted, 75)
		stats.IQR = stats.Q3 - stats.Q1
	}

	// === 自相关和周期性（需要单独计算）===
	stats.AutoCorrelation = computeAutoCorrelation(src, 10)
	stats.Periodicity, stats.PeriodicScore = detectPeriodicity(src)

	return stats
}

// computeBasicStats 计算基本统计量
func computeBasicStats(src []float64, stats *TimeSeriesStats) {
	sum := 0.0
	validCount := 0

	stats.Min = math.Inf(1)
	stats.Max = math.Inf(-1)

	// 第一遍：计算min, max, sum
	for _, v := range src {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		validCount++
		sum += v
		if v < stats.Min {
			stats.Min = v
		}
		if v > stats.Max {
			stats.Max = v
		}
	}

	if validCount == 0 {
		return
	}

	stats.Range = stats.Max - stats.Min
	stats.Mean = sum / float64(validCount)

	// 第二遍：计算方差、标准差
	sumSqDiff := 0.0
	sumCubeDiff := 0.0
	sumQuadDiff := 0.0

	for _, v := range src {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		diff := v - stats.Mean
		sumSqDiff += diff * diff
		sumCubeDiff += diff * diff * diff
		sumQuadDiff += diff * diff * diff * diff
	}

	stats.Variance = sumSqDiff / float64(validCount)
	stats.StdDev = math.Sqrt(stats.Variance)

	// 偏度和峰度
	if stats.StdDev > 0 {
		stats.Skewness = (sumCubeDiff / float64(validCount)) / math.Pow(stats.StdDev, 3)
		stats.Kurtosis = (sumQuadDiff/float64(validCount))/math.Pow(stats.StdDev, 4) - 3
	}

	// 计算中位数和四分位数
	sorted := make([]float64, 0, validCount)
	for _, v := range src {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			sorted = append(sorted, v)
		}
	}
	sort.Float64s(sorted)

	if len(sorted) > 0 {
		stats.Median = percentile(sorted, 50)
		stats.Q1 = percentile(sorted, 25)
		stats.Q3 = percentile(sorted, 75)
		stats.IQR = stats.Q3 - stats.Q1
		stats.PercentileP5 = percentile(sorted, 5)
		stats.PercentileP95 = percentile(sorted, 95)
	}
}

// computeValueProperties 计算数值特性
func computeValueProperties(src []float64, stats *TimeSeriesStats) {
	uniqueMap := make(map[float64]bool)

	for _, v := range src {
		uniqueMap[v] = true

		if v == 0 {
			stats.ZeroCount++
		}

		if v == math.Floor(v) {
			stats.IntegerCount++
		}
	}

	stats.UniqueCount = len(uniqueMap)
	stats.UniqueRatio = float64(stats.UniqueCount) / float64(len(src))
	stats.ZeroRatio = float64(stats.ZeroCount) / float64(len(src))
	stats.IntegerRatio = float64(stats.IntegerCount) / float64(len(src))
}

// computeDifferenceStats 计算差分统计
func computeDifferenceStats(src []float64, order int) *DifferenceStats {
	if len(src) <= order {
		return nil
	}

	diffs := make([]float64, len(src)-order)
	for i := order; i < len(src); i++ {
		diff := src[i]
		for j := 1; j <= order; j++ {
			diff -= src[i-j]
		}
		diffs[i-order] = diff
	}

	ds := &DifferenceStats{
		Min: math.Inf(1),
		Max: math.Inf(-1),
	}

	sum := 0.0
	zeroCount := 0
	uniqueMap := make(map[float64]bool)
	for _, d := range diffs {
		uniqueMap[d] = true
		if d < ds.Min {
			ds.Min = d
		}
		if d > ds.Max {
			ds.Max = d
		}
		sum += d
		if math.Abs(d) < 1e-10 {
			zeroCount++
		}
	}
	ds.UniqueCount = len(uniqueMap)
	ds.UniqueRatio = float64(ds.UniqueCount) / float64(len(src))
	ds.Mean = sum / float64(len(diffs))
	ds.Range = ds.Max - ds.Min
	ds.ZeroRatio = float64(zeroCount) / float64(len(diffs))

	// 计算标准差
	sumSqDiff := 0.0
	for _, d := range diffs {
		diff := d - ds.Mean
		sumSqDiff += diff * diff
	}
	ds.StdDev = math.Sqrt(sumSqDiff / float64(len(diffs)))

	return ds
}

// computeTimeSeriesFeatures 计算时序特征
func computeTimeSeriesFeatures(src []float64, stats *TimeSeriesStats) {
	if len(src) < 2 {
		return
	}

	// 单调性
	increasing := 0
	decreasing := 0
	changePoints := 0
	prevTrend := 0 // -1: decreasing, 0: equal, 1: increasing

	for i := 1; i < len(src); i++ {
		if math.IsNaN(src[i]) || math.IsNaN(src[i-1]) {
			continue
		}

		currentTrend := 0
		if src[i] > src[i-1] {
			increasing++
			currentTrend = 1
		} else if src[i] < src[i-1] {
			decreasing++
			currentTrend = -1
		}

		// 检测变化点
		if prevTrend != 0 && currentTrend != 0 && prevTrend != currentTrend {
			changePoints++
		}
		if currentTrend != 0 {
			prevTrend = currentTrend
		}
	}

	total := increasing + decreasing
	if total > 0 {
		stats.Monotonicity = float64(increasing-decreasing) / float64(total)
	}
	stats.ChangePoints = changePoints
}

// computeRunLengthStats 计算游程统计
func computeRunLengthStats(src []float64) *RunLengthStats {
	if len(src) == 0 {
		return nil
	}

	rls := &RunLengthStats{}
	currentRun := 1
	totalRun := 0
	constantRuns := 0
	singleCount := 0
	for i := 1; i < len(src); i++ {
		if math.IsNaN(src[i]) || math.IsNaN(src[i-1]) {
			continue
		}

		if src[i] == src[i-1] {
			currentRun++
		} else {
			if currentRun == 1 {
				singleCount++
			} else {
				constantRuns++
			}
			if currentRun > rls.MaxRunLength {
				rls.MaxRunLength = currentRun
			}
			totalRun += currentRun
			rls.RunCount++
			currentRun = 1
		}
	}

	// 处理最后一个游程
	if currentRun > 1 {
		constantRuns++
	}
	if currentRun > rls.MaxRunLength {
		rls.MaxRunLength = currentRun
	}
	totalRun += currentRun
	rls.RunCount++

	if rls.RunCount > 0 {
		rls.AvgRunLength = float64(totalRun) / float64(rls.RunCount)
		rls.ConstantRunRatio = float64(constantRuns) / float64(rls.RunCount)
		rls.InRunRatio = 1.0 - float64(singleCount)/float64(len(src))
	}

	return rls
}

// computeBitLevelStats 计算位级统计
func computeBitLevelStats(src []float64) *BitLevelStats {
	bls := &BitLevelStats{}

	if len(src) == 0 {
		return bls
	}

	signChanges := 0
	totalSetBits := 0
	exponentMap := make(map[int]int)

	for i, v := range src {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}

		// 符号变化
		if i > 0 && !math.IsNaN(src[i-1]) {
			if (v < 0 && src[i-1] >= 0) || (v >= 0 && src[i-1] < 0) {
				signChanges++
			}
		}

		// 位统计
		bits := math.Float64bits(v)
		totalSetBits += popcount(bits)

		// 提取指数
		exponent := int((bits>>52)&0x7FF) - 1023
		exponentMap[exponent]++
	}

	bls.SignChanges = signChanges
	bls.AvgSetBits = float64(totalSetBits) / float64(len(src))

	// 计算指数范围和最常见指数
	minExp, maxExp := math.MaxInt32, math.MinInt32
	maxCount := 0
	for exp, count := range exponentMap {
		if exp < minExp {
			minExp = exp
		}
		if exp > maxExp {
			maxExp = exp
		}
		if count > maxCount {
			maxCount = count
			bls.CommonExponent = exp
		}
	}
	bls.ExponentRange = maxExp - minExp

	// 计算尾数熵（简化版）
	mantissaMap := make(map[uint64]int)
	for _, v := range src {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			bits := math.Float64bits(v)
			mantissa := bits & 0xFFFFFFFFFFFFF
			mantissaMap[mantissa]++
		}
	}
	bls.MantissaEntropy = calculateMapEntropy(mantissaMap, len(src))

	return bls
}

// computeEntropy 计算信息熵
func computeEntropy(src []float64) float64 {
	if len(src) == 0 {
		return 0
	}

	// 使用直方图方法计算熵
	bins := 100
	min, max := math.Inf(1), math.Inf(-1)

	// 找出有效范围
	validCount := 0
	for _, v := range src {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
			validCount++
		}
	}

	if validCount == 0 || min == max {
		return 0
	}

	// 创建直方图
	hist := make([]int, bins)
	binWidth := (max - min) / float64(bins)

	for _, v := range src {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			binIdx := int((v - min) / binWidth)
			if binIdx >= bins {
				binIdx = bins - 1
			}
			hist[binIdx]++
		}
	}

	// 计算熵
	entropy := 0.0
	for _, count := range hist {
		if count > 0 {
			p := float64(count) / float64(validCount)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// computeAutoCorrelation 计算自相关系数
func computeAutoCorrelation(src []float64, maxLag int) []float64 {
	if len(src) < maxLag+1 {
		return nil
	}

	// 计算均值
	mean := 0.0
	validCount := 0
	for _, v := range src {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			mean += v
			validCount++
		}
	}
	if validCount == 0 {
		return nil
	}
	mean /= float64(validCount)

	// 计算方差
	variance := 0.0
	for _, v := range src {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			diff := v - mean
			variance += diff * diff
		}
	}
	variance /= float64(validCount)

	if variance == 0 {
		return nil
	}

	// 计算各个lag的自相关系数
	correlations := make([]float64, maxLag)
	for lag := 1; lag <= maxLag; lag++ {
		covariance := 0.0
		count := 0

		for i := lag; i < len(src); i++ {
			if !math.IsNaN(src[i]) && !math.IsNaN(src[i-lag]) &&
				!math.IsInf(src[i], 0) && !math.IsInf(src[i-lag], 0) {
				covariance += (src[i] - mean) * (src[i-lag] - mean)
				count++
			}
		}

		if count > 0 {
			covariance /= float64(count)
			correlations[lag-1] = covariance / variance
		}
	}

	return correlations
}

// detectPeriodicity 检测周期性
func detectPeriodicity(src []float64) (int, float64) {
	if len(src) < 10 {
		return 0, 0
	}

	// 使用自相关检测周期
	maxLag := min(len(src)/2, 100)
	correlations := computeAutoCorrelation(src, maxLag)

	if correlations == nil {
		return 0, 0
	}

	// 找出第一个显著的峰值
	threshold := 0.5
	for i := 1; i < len(correlations)-1; i++ {
		// 检测局部最大值
		if correlations[i] > threshold &&
			correlations[i] > correlations[i-1] &&
			correlations[i] > correlations[i+1] {
			return i + 1, correlations[i]
		}
	}

	return 0, 0
}

// 辅助函数

// percentile 计算百分位数
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	rank := p / 100 * float64(len(sorted)-1)
	lowerIdx := int(math.Floor(rank))
	upperIdx := int(math.Ceil(rank))

	if lowerIdx == upperIdx {
		return sorted[lowerIdx]
	}

	weight := rank - float64(lowerIdx)
	return sorted[lowerIdx]*(1-weight) + sorted[upperIdx]*weight
}

// popcount 计算设置位数
func popcount(x uint64) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// calculateMapEntropy 计算map的熵
func calculateMapEntropy(m map[uint64]int, total int) float64 {
	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, count := range m {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// min 返回较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
