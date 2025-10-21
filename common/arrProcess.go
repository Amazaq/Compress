package common

import "math"

func DeltaArr(src []float64) []float64 {
	if len(src) < 2 {
		return src
	}

	for i := len(src) - 1; i > 0; i-- {
		src[i] = src[i] - src[i-1]
	}

	return src
}
func DeltaRecover(src []float64) []float64 {
	if len(src) < 2 {
		return src
	}

	for i := 1; i < len(src); i++ {
		src[i] = src[i] + src[i-1]
	}

	return src
}
func DeltaOfDeltaArr(src []float64) []float64 {
	src = DeltaArr(src)
	src = DeltaArr(src)
	return src
}
func DeltaOfDeltaRecover(src []float64) []float64 {
	src = DeltaRecover(src)
	src = DeltaRecover(src)
	return src
}
func RangedArr(src []float64) ([]float64, float64) {
	if len(src) < 2 {
		return src, 0
	}

	minNum := src[0]
	for _, val := range src {
		if val < minNum {
			minNum = val
		}
	}

	for i, _ := range src {
		src[i] -= minNum
	}
	return src, minNum
}
func LogedArr(src []float64) ([]float64, float64) {
	if len(src) < 2 {
		return src, 0
	}

	minNum := src[0]
	for _, val := range src {
		if val < minNum {
			minNum = val
		}
	}
	offset := 1.0 - minNum // 如果有负值，需要偏移
	result := make([]float64, len(src))
	for i, v := range src {
		result[i] = math.Log(v + offset)
	}
	return result, offset
}
func LogedRecover(src []float64, offset float64) []float64 {
	for i := 0; i < len(src); i++ {
		src[i] = math.Exp(src[i]) - offset
	}
	return src
}
func RangedRecover(src []float64, base float64) []float64 {
	for i := 0; i < len(src); i++ {
		src[i] += base
	}
	return src
}
func MinMaxRangedArr(src []float64) ([]float64, float64, float64) {
	n := len(src)
	if n == 0 {
		return nil, 0, 0
	}

	minNum := math.MaxFloat64
	maxNum := -math.MaxFloat64

	// 1. 先找出最小值和最大值
	for _, v := range src {
		if v < minNum {
			minNum = v
		}
		if v > maxNum {
			maxNum = v
		}
	}

	scaled := make([]float64, n)

	// 2. 如果所有值都相等，直接返回 0 数组，避免除以 0
	if maxNum == minNum {
		for i := range scaled {
			scaled[i] = 0
		}
		return scaled, minNum, maxNum
	}

	scale := maxNum - minNum
	// 3. 缩放到 [0,1]
	for i, v := range src {
		scaled[i] = (v - minNum) / scale
	}

	return scaled, minNum, maxNum
}
func MinMaxRangedRecover(src []float64, minNum, maxNum float64) []float64 {
	n := len(src)
	recovered := make([]float64, n)

	// 如果 max=min，说明原数组是常数序列，直接填回常数
	if maxNum == minNum {
		for i := range recovered {
			recovered[i] = minNum
		}
		return recovered
	}

	scale := maxNum - minNum
	for i, v := range src {
		recovered[i] = v*scale + minNum
	}
	return recovered
}
func TransferToIntArr(src []float64) ([]int64, []int64) {
	return nil, nil
}

// ZScoreNormArr 对数组做 Z-Score 标准化，返回标准化后的新数组、均值和标准差（总体标准差）
func ZScoreNormArr(src []float64) ([]float64, float64, float64) {
	n := len(src)
	if n == 0 {
		return nil, 0, 0
	}

	// 计算均值
	var sum float64
	for _, v := range src {
		sum += v
	}
	mean := sum / float64(n)

	// 计算总体方差
	var sq float64
	for _, v := range src {
		d := v - mean
		sq += d * d
	}
	var std float64
	if sq == 0 {
		std = 0
	} else {
		std = math.Sqrt(sq / float64(n))
	}

	norm := make([]float64, n)
	if std == 0 {
		// 常数序列，Z-Score 全为 0
		for i := range norm {
			norm[i] = 0
		}
		return norm, mean, 0
	}

	inv := 1.0 / std
	for i, v := range src {
		norm[i] = (v - mean) * inv
	}
	return norm, mean, std
}

// ZScoreNormRecover 将 Z-Score 标准化后的数组还原
func ZScoreNormRecover(src []float64, mean, std float64) []float64 {
	n := len(src)
	recovered := make([]float64, n)
	if std == 0 {
		for i := range recovered {
			recovered[i] = mean
		}
		return recovered
	}
	for i, z := range src {
		recovered[i] = z*std + mean
	}
	return recovered
}
