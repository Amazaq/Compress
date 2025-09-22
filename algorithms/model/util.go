package model

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"myalgo/algorithms/rule"
	"os"
)

func newCSVWriter(path string) *csv.Writer {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	return csv.NewWriter(f)
}
func flattenStats(s *rule.TimeSeriesStats) []float64 {
	var v []float64
	// 基本统计
	v = append(v, s.Min, s.Max, s.Mean, s.Median, s.StdDev, s.Variance,
		s.Skewness, s.Kurtosis, s.Range, s.IQR, s.Q1, s.Q3)

	// 数值特性
	v = append(v,
		float64(s.UniqueCount), s.UniqueRatio,
		float64(s.ZeroCount), s.ZeroRatio,
		float64(s.IntegerCount), s.IntegerRatio,
	)

	// 差分统计
	if s.DiffStats != nil {
		v = append(v, s.DiffStats.Min, s.DiffStats.Max, s.DiffStats.Mean,
			s.DiffStats.StdDev, s.DiffStats.Range,
			s.DiffStats.ZeroRatio, float64(s.DiffStats.UniqueCount), s.DiffStats.UniqueRatio)
	} else {
		v = append(v, make([]float64, 8)...)
	}

	if s.SecondDiffStats != nil {
		v = append(v, s.SecondDiffStats.Min, s.SecondDiffStats.Max, s.SecondDiffStats.Mean,
			s.SecondDiffStats.StdDev, s.SecondDiffStats.Range,
			s.SecondDiffStats.ZeroRatio, float64(s.SecondDiffStats.UniqueCount), s.SecondDiffStats.UniqueRatio)
	} else {
		v = append(v, make([]float64, 8)...)
	}

	// 时序特征
	v = append(v, s.Monotonicity, s.Smoothness, float64(s.ChangePoints))

	// 游程统计
	if s.RunLength != nil {
		v = append(v,
			float64(s.RunLength.MaxRunLength),
			s.RunLength.AvgRunLength,
			float64(s.RunLength.RunCount),
			s.RunLength.ConstantRunRatio)
	} else {
		v = append(v, make([]float64, 4)...)
	}

	// 位级统计
	if s.BitStats != nil {
		v = append(v,
			s.BitStats.AvgSetBits,
			float64(s.BitStats.SignChanges),
			s.BitStats.MantissaEntropy,
			float64(s.BitStats.ExponentRange),
			float64(s.BitStats.CommonExponent))
	} else {
		v = append(v, make([]float64, 5)...)
	}

	// 分布特征
	v = append(v, s.Entropy, s.PercentileP95, s.PercentileP5)

	// 自相关
	v = append(v, s.AutoCorrelation...)

	// 周期特征
	v = append(v, float64(s.Periodicity), s.PeriodicScore)

	return v
}
func nullFunc(src []float64) []float64 {
	return src
}
func nullRangedFunc(src []float64) ([]float64, float64) {
	return src, 0
}
func nullRangedRecoverFunc(src []float64, base float64) []float64 {
	return src
}
func RunCompressWithParam(dst []byte, src []float64, param []int) []byte {
	diffed := delFunc[param[0]].transfer(src)
	subbed, base := rangedFunc[param[1]].transfer(diffed)
	dst = append(dst, byte(param[0]))
	dst = append(dst, byte(param[1]))
	dst = append(dst, byte(param[2]))
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(base))
	dst = compressFunc[param[2]].compress(dst, subbed)

	return dst
}
func RunDecompress(dst []float64, src []byte) ([]float64, error) {
	if len(src) < 11 {
		return nil, fmt.Errorf("invalid data: byte slice is too short, received %d bytes, need at least 11", len(src))
	}

	param0 := int(src[0])
	param1 := int(src[1])
	param2 := int(src[2])

	if param0 >= len(delFunc) || param1 >= len(rangedFunc) || param2 >= len(compressFunc) {
		return nil, fmt.Errorf("invalid parameters in data stream: p0=%d, p1=%d, p2=%d", param0, param1, param2)
	}
	offset := 3
	baseBits := binary.LittleEndian.Uint64(src[offset : offset+8])
	base := math.Float64frombits(baseBits)
	offset += 8

	decompressor := compressFunc[param2].decompress
	dst, err := decompressor(dst, src[offset:])
	if err != nil {
		return nil, fmt.Errorf("decompression failed using %s: %w", compressFunc[param2].algo, err)
	}

	rangedReverser := rangedFunc[param1].reverse
	dst = rangedReverser(dst, base)
	deltaReverser := delFunc[param0].reverse
	dst = deltaReverser(dst)

	return dst, nil
}
