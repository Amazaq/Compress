package rle

// 游程编码实现
func RunLengthEncode(src []uint64) []uint64 {
	if len(src) == 0 {
		return nil
	}

	result := make([]uint64, 0)
	currentVal := src[0]
	count := uint64(1)

	for i := 1; i < len(src); i++ {
		if src[i] == currentVal {
			count++
		} else {
			// 存储当前值和计数
			result = append(result, currentVal, count)
			currentVal = src[i]
			count = 1
		}
	}

	// 添加最后一组
	result = append(result, currentVal, count)

	return result
}

// 游程解码实现
func RunLengthDecode(src []uint64, length int) []uint64 {
	if len(src) == 0 || len(src)%2 != 0 {
		return nil
	}

	result := make([]uint64, 0, length)

	for i := 0; i < len(src); i += 2 {
		val := src[i]
		count := src[i+1]

		for j := uint64(0); j < count; j++ {
			result = append(result, val)
		}
	}

	return result
}
