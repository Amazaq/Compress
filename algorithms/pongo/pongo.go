package pongo

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// reverseBits 对 uint64 进行位反转
func reverseBits(n uint64) uint64 {
	var result uint64
	for i := 0; i < 64; i++ {
		result = (result << 1) | (n & 1)
		n >>= 1
	}
	return result
}

// Eraser 将浮点数的小数部分用整数二进制表示替换 IEEE754 尾数
// 根据指数计算哪些位是整数部分，哪些位是小数部分
// exp = exponent - 1023, 前 12+exp 位保留(符号+指数+整数尾数)，后面替换为小数的整数表示
func Eraser(value float64) float64 {
	// 获取原始二进制
	originalBits := math.Float64bits(value)

	// 提取指数部分 (11位)
	exponentBits := (originalBits >> 52) & 0x7FF
	exp := int(exponentBits) - 1023 // 实际指数

	// 分离整数和小数部分
	intPart := math.Floor(math.Abs(value))
	fracPart := math.Abs(value) - intPart

	// 将小数部分转换为整数
	// 例如：0.18 -> "18" -> 整数 18
	// 使用字符串方式提取小数部分，避免浮点精度问题
	fracInt := uint64(0)

	// 将浮点数转换为字符串，提取小数点后的数字
	str := strconv.FormatFloat(fracPart, 'f', -1, 64)
	parts := strings.Split(str, ".")
	if len(parts) == 2 {
		// 将小数部分字符串转换为整数
		fracInt, _ = strconv.ParseUint(parts[1], 10, 64)
	}

	// 计算需要保留和替换的位数
	// 前 12+exp 位：1位符号 + 11位指数 + exp位整数尾数
	// 后 52-exp 位：小数部分尾数，用 fracInt 替换
	preserveBits := 12 + exp // 符号位 + 指数位 + 整数部分尾数位
	replaceBits := 52 - exp  // 小数部分尾数位

	// 对 fracInt 进行位反转
	// 例如：18 = 0b10010 -> reverse -> 0b01001000...
	reversedFrac := reverseBits(fracInt)

	// 保留前 preserveBits 位，后面用 reversedFrac 右移填充
	preserveMask := (uint64(0xFFFFFFFFFFFFFFFF) << uint(64-preserveBits))
	preservedPart := originalBits & preserveMask

	// 将 reversedFrac 右移 12+exp 位
	replacedPart := reversedFrac >> uint(preserveBits)

	erasedBits := preservedPart | replacedPart

	// 打印调试信息
	fmt.Printf("原始值: %.2f, exp=%d, 保留前%d位, 替换后%d位\n", value, exp, preserveBits, replaceBits)
	fmt.Printf("%064b:%f\n", originalBits, value)
	fmt.Printf("%064b:%f\n", erasedBits, math.Float64frombits(erasedBits))
	fmt.Printf("小数整数:   %d (二进制: %b)\n", fracInt, fracInt)
	fmt.Println()

	return math.Float64frombits(erasedBits)
}
