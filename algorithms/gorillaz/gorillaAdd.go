package gorillaz

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"myalgo/common"
	"os"
	"path/filepath"
)

// CompressAdd uses addition/subtraction instead of XOR
// It computes the difference between consecutive values
func CompressAdd(dst []byte, src []uint64) []byte {
	v := src[0]
	prev := v
	bs := &common.ByteWrapper{Stream: &dst, Count: 0}
	bs.AppendBits(v, 64) // append first value without any compression
	src = src[1:]
	prevLeadingZeros, prevTrailingZeros := ^uint8(0), uint8(0)
	sigbits := uint8(0)
	for _, num := range src {
		// Use addition instead of XOR
		v = num + prev

		if v == 0 {
			bs.AppendBit(common.Zero)
		} else {
			bs.AppendBit(common.One)
			leadingZeros, trailingZeros := uint8(bits.LeadingZeros64(v)), uint8(bits.TrailingZeros64(v))
			// clamp number of leading zeros to avoid overflow when encoding
			if leadingZeros >= 64 {
				leadingZeros = 63
			}
			if prevLeadingZeros != ^uint8(0) && leadingZeros >= prevLeadingZeros && trailingZeros >= prevTrailingZeros {
				bs.AppendBit(common.Zero)
				bs.AppendBits(v>>prevTrailingZeros, 64-int(prevLeadingZeros)-int(prevTrailingZeros))
			} else {
				prevLeadingZeros, prevTrailingZeros = leadingZeros, trailingZeros
				bs.AppendBit(common.One)
				bs.AppendBits(uint64(leadingZeros), 6)
				sigbits = 64 - leadingZeros - trailingZeros
				bs.AppendBits(uint64(sigbits), 6)
				bs.AppendBits(v>>trailingZeros, int(sigbits))
			}
		}
		prev = num
	}
	bs.Finish()
	return dst
}

// CompressFloatAdd uses addition instead of XOR for float64 values
// Encoding format:
//   - 2 bits for first bit and last bit combination (firstBit|lastBit)
//     00: first=0, last=0
//     01: first=0, last=1
//     10: first=1, last=0
//     11: first=1, last=1
//   - 6 bits for middle bits length
//   - middle bits (excluding first and last)
func CompressFloatAdd(dst []byte, src []float64) []byte {
	// 打开文件用于写入 ADD 结果
	addFile, err := openAddFile()
	if err == nil {
		defer addFile.Close()
	}

	v := math.Float64bits(src[0])
	prev := v
	bs := &common.ByteWrapper{Stream: &dst, Count: 0}
	bs.AppendBits(v, 64) // append first value without any compression

	// 写入第一个值
	if addFile != nil {
		fmt.Fprintf(addFile, "First value: %064b\n", v)
	}

	src = src[1:]
	for _, num := range src {
		curr := math.Float64bits(num)
		// Use addition instead of XOR
		v = curr + prev

		// 同时计算减法结果用于输出
		var diff uint64
		diff = curr - prev

		// 计算 XOR 结果
		xorResult := curr ^ prev

		// 写入 ADD、SUB 和 XOR 结果到文件
		if addFile != nil {
			fmt.Fprintf(addFile, "add=%064b xor=%064b sub=%064b\n", v, xorResult, diff)
		}

		if v == 0 {
			// 全0的情况，用1个bit表示
			bs.AppendBit(common.Zero)
		} else {
			bs.AppendBit(common.One)

			// 提取第一位和最后一位
			firstBit := (v >> 63) & 1 // 最高位（第63位）
			lastBit := v & 1          // 最低位（第0位）

			// 组合成2位模式: firstBit(1位) + lastBit(1位)
			pattern := (firstBit << 1) | lastBit
			bs.AppendBits(pattern, 2)

			// 找出中间62位（去掉最高位bit63和最低位bit0）的前导零和后导零
			middle62Bits := (v >> 1) & 0x3FFFFFFFFFFFFFFF // 提取bit1到bit62

			var leadingZeros, trailingZeros uint8

			if middle62Bits == 0 {
				// 中间62位全是0
				leadingZeros = 62
				trailingZeros = 0 // 或者可以设为62，但这里设为0以便计算sigBits=0
			} else {
				leadingZeros = uint8(bits.LeadingZeros64(middle62Bits))
				trailingZeros = uint8(bits.TrailingZeros64(middle62Bits))

				// 调整，因为我们只看62位（middle62Bits最高位是bit61，对应原始的bit62）
				// bits.LeadingZeros64会计算64位的前导零，需要减去高2位
				if leadingZeros >= 2 {
					leadingZeros -= 2
				}

				// 确保不超过62
				if leadingZeros > 62 {
					leadingZeros = 62
				}
				if trailingZeros > 62 {
					trailingZeros = 62
				}
			}

			// 写入前导零数量（6位，最大62）
			bs.AppendBits(uint64(leadingZeros), 6)

			// 写入后导零数量（6位，最大62）
			bs.AppendBits(uint64(trailingZeros), 6)

			// 计算中间有效位的长度
			var sigBits uint8
			if leadingZeros+trailingZeros < 62 {
				sigBits = 62 - leadingZeros - trailingZeros
			} else {
				sigBits = 0
			}

			// 写入中间的有效位
			if sigBits > 0 && middle62Bits != 0 {
				mask := (uint64(1) << sigBits) - 1
				middleBits := (middle62Bits >> trailingZeros) & mask
				bs.AppendBits(middleBits, int(sigBits))
			}
		}
		prev = curr
	}
	bs.Finish()
	return dst
}

// openAddFile opens or creates the add.txt file in dataset directory
func openAddFile() (*os.File, error) {
	// 获取当前工作目录的上级目录
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// 构建 dataset/add.txt 路径
	// 尝试多个可能的路径
	possiblePaths := []string{
		filepath.Join(dir, "dataset", "add.txt"),
		filepath.Join(dir, "..", "..", "dataset", "add.txt"),
		filepath.Join(dir, "..", "..", "..", "dataset", "add.txt"),
	}

	for _, path := range possiblePaths {
		// 确保 dataset 目录存在
		datasetDir := filepath.Dir(path)
		if err := os.MkdirAll(datasetDir, 0755); err == nil {
			// 打开或创建文件
			file, err := os.Create(path)
			if err == nil {
				return file, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to create add.txt file")
}

// DecompressAdd uses addition instead of XOR to reconstruct values
func DecompressAdd(dst []uint64, src []byte) ([]uint64, error) {
	bs := &common.ByteWrapper{Stream: &src, Count: 8}
	firstValue, err := bs.ReadBits(64)
	if err != nil {
		return nil, err
	}
	dst = append(dst, firstValue)
	prev := firstValue
	prevLeadingZeros, prevTrailingZeros := uint8(0), uint8(0)
	for true {
		b, err := bs.ReadBit()
		if err != nil {
			return nil, err
		}
		if b == common.Zero {
			dst = append(dst, prev)
			continue
		} else {
			b, err = bs.ReadBit()
			if err != nil {
				return nil, err
			}
			leadingZeros, trailingZeros := prevLeadingZeros, prevTrailingZeros
			if b == common.One {
				bts, err := bs.ReadBits(6) // read leading zeros' length
				if err != nil {
					return nil, err
				}
				leadingZeros = uint8(bts)
				bts, err = bs.ReadBits(6) // read sig's length
				if err != nil {
					return nil, err
				}
				midLen := uint8(bts)
				if midLen == 0 {
					midLen = 64
				}
				if midLen+leadingZeros > 64 {
					if b, err = bs.ReadBit(); b == common.Zero {
						return dst, nil
					}
					return nil, errors.New("invalid bits")
				}
				trailingZeros = 64 - leadingZeros - midLen
				prevLeadingZeros, prevTrailingZeros = leadingZeros, trailingZeros
			}
			bts, err := bs.ReadBits(int(64 - leadingZeros - trailingZeros))
			if err != nil {
				return nil, err
			}
			addResult := bts << trailingZeros

			// Use subtraction instead of XOR (inverse of addition during compression)
			v := addResult - prev

			dst = append(dst, v)
			prev = v
		}
	}
	return dst, nil
}

// DecompressFloatAdd uses addition instead of XOR to reconstruct float64 values
// Strictly follows the inverse of CompressFloatAdd
func DecompressFloatAdd(dst []float64, src []byte) ([]float64, error) {
	bs := &common.ByteWrapper{Stream: &src, Count: 8}

	// 读取第一个值（64位，未压缩）
	firstValue, err := bs.ReadBits(64)
	if err != nil {
		return nil, err
	}
	dst = append(dst, math.Float64frombits(firstValue))
	prev := firstValue

	// 循环读取后续的压缩值
	for {
		// 读取1位标识：0表示ADD结果为0，1表示非零
		b, err := bs.ReadBit()
		if err != nil {
			// 到达数据末尾，正常返回
			return dst, nil
		}

		if b == common.Zero {
			// ADD结果为0，说明 curr + prev = 0，所以 curr = -prev
			// 但在无符号整数运算中，curr = 0 - prev = ^prev + 1（补码）
			// 实际上，如果v=0，说明curr + prev = 0（模2^64）
			// 所以curr = -prev = 2^64 - prev（在uint64中）
			currBits := (^prev) + 1
			dst = append(dst, math.Float64frombits(currBits))
			prev = currBits
			continue
		}

		// ADD结果非零，读取编码的位模式

		// 读取2位：firstBit|lastBit
		patternBits, err := bs.ReadBits(2)
		if err != nil {
			return dst, nil
		}
		firstBit := (patternBits >> 1) & 1
		lastBit := patternBits & 1

		// 读取6位：中间62位的前导零数量
		leadingZerosBits, err := bs.ReadBits(6)
		if err != nil {
			return dst, nil
		}
		leadingZeros := uint8(leadingZerosBits)

		// 读取6位：中间62位的后导零数量
		trailingZerosBits, err := bs.ReadBits(6)
		if err != nil {
			return dst, nil
		}
		trailingZeros := uint8(trailingZerosBits)

		// 计算有效位数量
		var sigBits uint8
		if leadingZeros+trailingZeros < 62 {
			sigBits = 62 - leadingZeros - trailingZeros
		} else {
			sigBits = 0
		}

		// 重建middle62Bits（bit1到bit62）
		var middle62Bits uint64 = 0
		if sigBits > 0 {
			// 读取有效位
			middleBits, err := bs.ReadBits(int(sigBits))
			if err != nil {
				return dst, nil
			}
			// 将有效位放回正确的位置
			middle62Bits = middleBits << trailingZeros
		}

		// 重建完整的64位ADD结果
		var addResult uint64 = 0

		// 设置bit63（最高位）
		addResult |= (firstBit << 63)

		// 设置bit1到bit62（中间62位）
		addResult |= (middle62Bits << 1)

		// 设置bit0（最低位）
		addResult |= lastBit

		// 恢复原始值：addResult = curr + prev，所以 curr = addResult - prev
		currBits := addResult - prev

		dst = append(dst, math.Float64frombits(currBits))
		prev = currBits
	}
}
