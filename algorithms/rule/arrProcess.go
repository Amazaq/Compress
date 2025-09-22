package rule

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
func RangedRecover(src []float64, base float64) []float64 {
	for i := 0; i < len(src); i++ {
		src[i] += base
	}
	return src
}
func TransferToIntArr(src []float64) ([]int64, []int64) {
	return nil, nil
}
