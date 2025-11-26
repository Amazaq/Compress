package gorillaz

import (
	"fmt"
	"math"
	"myalgo/algorithms/elf"
)

// XorWithAndWithoutErase performs XOR with and without erasure, writing results to add.txt
// 1. Direct XOR: curr ^ prev
// 2. Erase then XOR: erased_curr ^ erased_prev (using ELF erasure logic)
func XorWithAndWithoutErase(src []float64) error {
	if len(src) < 2 {
		return fmt.Errorf("need at least 2 values")
	}

	// 打开 add.txt 文件
	addFile, err := openAddFile()
	if err != nil {
		return fmt.Errorf("failed to open add.txt: %v", err)
	}
	defer addFile.Close()

	// 写入标题
	fmt.Fprintf(addFile, "=== XOR with and without Erasure ===\n\n")
	fmt.Fprintf(addFile, "First value: %064b\n\n", math.Float64bits(src[0]))

	// 初始化
	prevBits := math.Float64bits(src[0])
	prevErased := prevBits
	lastBetaStar := math.MaxInt32

	for i := 1; i < len(src); i++ {
		v := src[i]
		currBits := math.Float64bits(v)

		// 1. 直接 XOR
		directXor := currBits ^ prevBits

		// 2. 擦除后 XOR
		var currErased uint64

		// 按照 ELF 的擦除逻辑
		if v == 0.0 || math.IsInf(v, 0) || math.IsNaN(v) {
			// 特殊值不擦除
			currErased = currBits
		} else {
			alpha, betaStar := elf.GetAlphaAndBetaStar(v, lastBetaStar)
			e := int((currBits >> 52) & 0x7ff)
			gAlpha := elf.GetFAlpha(alpha) + e - 1023
			eraseBits := 52 - gAlpha

			if eraseBits > 4 {
				// 执行擦除
				mask := ^uint64(0) << uint(eraseBits)
				delta := (^mask) & currBits

				if delta != 0 {
					currErased = mask & currBits
					lastBetaStar = betaStar
				} else {
					currErased = currBits
				}
			} else {
				currErased = currBits
			}
		}

		erasedXor := currErased ^ prevErased

		// 写入结果
		fmt.Fprintf(addFile, "[%d] [%f] direct_xor=%064b erased_xor=%064b\n", i, v, directXor, erasedXor)

		prevBits = currBits
		prevErased = currErased
	}

	fmt.Fprintf(addFile, "\n=== Total: %d values processed ===\n", len(src))

	return nil
}
