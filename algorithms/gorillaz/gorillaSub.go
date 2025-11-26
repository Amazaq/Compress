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

// CompressFloatSub uses subtraction instead of XOR for float64 values
// Encoding format is the same as original Gorilla:
//   - If v==0: 1 bit (0)
//   - If v!=0: 1 bit (1) + control bit + value bits
//   - Control bit 0: use previous leading/trailing zeros
//   - Control bit 1: 6 bits leading + 6 bits sigbits + sigbits data
func CompressFloatSub(dst []byte, src []float64) []byte {
	// 打开文件用于写入位数统计
	bitsFile, err := openBitsFile()
	if err == nil {
		defer bitsFile.Close()
	}

	// 打开文件用于写入二进制结果
	addFile, err := openAddFile()
	if err == nil {
		defer addFile.Close()
	}

	v := math.Float64bits(src[0])
	prev := v
	bs := &common.ByteWrapper{Stream: &dst, Count: 0}
	bs.AppendBits(v, 64) // append first value without any compression

	// 写入第一个值到 add.txt
	if addFile != nil {
		fmt.Fprintf(addFile, "First value: %064b\n", v)
	}

	src = src[1:]
	prevLeadingZeros, prevTrailingZeros := ^uint8(0), uint8(0)
	prevLeadingZerosXor, prevTrailingZerosXor := ^uint8(0), uint8(0)
	sigbits := uint8(0)

	for i, num := range src {
		curr := math.Float64bits(num)

		// Use subtraction instead of XOR
		// 直接计算差值（有符号，通过补码表示）
		v = curr - prev

		// 同时计算 XOR 结果用于统计
		xorResult := curr ^ prev

		// 计算 SUB 需要的位数
		var subBits int
		if v == 0 {
			subBits = 1 // 只需要1位表示0
			bs.AppendBit(common.Zero)
		} else {
			bs.AppendBit(common.One)
			leadingZeros, trailingZeros := uint8(bits.LeadingZeros64(v)), uint8(bits.TrailingZeros64(v))
			// clamp number of leading zeros to avoid overflow when encoding
			if leadingZeros >= 64 {
				leadingZeros = 63
			}
			if prevLeadingZeros != ^uint8(0) && leadingZeros >= prevLeadingZeros && trailingZeros >= prevTrailingZeros {
				// 使用前一个 leading/trailing zeros: 1bit(非零) + 1bit(控制) + 有效位
				subBits = 1 + 1 + (64 - int(prevLeadingZeros) - int(prevTrailingZeros))
				bs.AppendBit(common.Zero)
				bs.AppendBits(v>>prevTrailingZeros, 64-int(prevLeadingZeros)-int(prevTrailingZeros))
			} else {
				prevLeadingZeros, prevTrailingZeros = leadingZeros, trailingZeros
				sigbits = 64 - leadingZeros - trailingZeros
				// 1bit(非零) + 1bit(控制) + 6bits(leading) + 6bits(sigbits) + sigbits
				subBits = 1 + 1 + 6 + 6 + int(sigbits)
				bs.AppendBit(common.One)
				bs.AppendBits(uint64(leadingZeros), 6)
				bs.AppendBits(uint64(sigbits), 6)
				bs.AppendBits(v>>trailingZeros, int(sigbits))
			}
		}

		// 计算 XOR 需要的位数（使用相同的 Gorilla 编码逻辑）
		var xorBits int
		if xorResult == 0 {
			xorBits = 1
		} else {
			leadingZerosXor, trailingZerosXor := uint8(bits.LeadingZeros64(xorResult)), uint8(bits.TrailingZeros64(xorResult))
			if leadingZerosXor >= 64 {
				leadingZerosXor = 63
			}
			if prevLeadingZerosXor != ^uint8(0) && leadingZerosXor >= prevLeadingZerosXor && trailingZerosXor >= prevTrailingZerosXor {
				// 使用前一个 leading/trailing zeros
				xorBits = 1 + 1 + (64 - int(prevLeadingZerosXor) - int(prevTrailingZerosXor))
			} else {
				prevLeadingZerosXor, prevTrailingZerosXor = leadingZerosXor, trailingZerosXor
				sigbitsXor := 64 - leadingZerosXor - trailingZerosXor
				xorBits = 1 + 1 + 6 + 6 + int(sigbitsXor)
			}
		}

		// 写入位数统计到 bits.txt
		if bitsFile != nil {
			fmt.Fprintf(bitsFile, "%d: xor(%d) sub(%d)\n", i, xorBits, subBits)
		}

		// 写入二进制结果到 add.txt
		if addFile != nil {
			fmt.Fprintf(addFile, "xor=%064b sub=%064b\n", xorResult, v)
		}

		prev = curr
	}
	bs.Finish()
	return dst
}

// DecompressFloatSub reconstructs float64 values from subtraction-based compression
func DecompressFloatSub(dst []float64, src []byte) ([]float64, error) {
	bs := &common.ByteWrapper{Stream: &src, Count: 8}
	firstValue, err := bs.ReadBits(64)
	if err != nil {
		return nil, err
	}
	dst = append(dst, math.Float64frombits(firstValue))
	prev := firstValue
	prevLeadingZeros, prevTrailingZeros := uint8(0), uint8(0)

	for {
		b, err := bs.ReadBit()
		if err != nil {
			return nil, err
		}
		if b == common.Zero {
			dst = append(dst, math.Float64frombits(prev))
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

			// 重建差值
			diff := bts << trailingZeros

			// 对于减法压缩：curr - prev = diff，所以 curr = prev + diff
			// Go的uint64减法自动处理补码，加法即可恢复
			v := prev + diff

			dst = append(dst, math.Float64frombits(v))
			prev = v
		}
	}
	return dst, nil
}

// openBitsFile opens or creates the bits.txt file in dataset directory
func openBitsFile() (*os.File, error) {
	// 获取当前工作目录的上级目录
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// 构建 dataset/bits.txt 路径
	// 尝试多个可能的路径
	possiblePaths := []string{
		filepath.Join(dir, "dataset", "bits.txt"),
		filepath.Join(dir, "..", "..", "dataset", "bits.txt"),
		filepath.Join(dir, "..", "..", "..", "dataset", "bits.txt"),
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

	return nil, fmt.Errorf("failed to create bits.txt file")
}
