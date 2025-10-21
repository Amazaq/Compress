package model

import (
	"encoding/csv"
	"log"
	"myalgo/algorithms/chimp128"
	"myalgo/algorithms/elf"
	"myalgo/algorithms/fpc"
	"myalgo/algorithms/huffman"
	"myalgo/algorithms/zstd"
	"myalgo/common"
	"os"
)

var rangedFunc = []struct {
	ranged   string
	transfer func(src []float64) ([]float64, float64)
	reverse  func(src []float64, base float64) []float64
}{
	{"null", nullRangedFunc, nullRangedRecoverFunc},
	{"log", common.LogedArr, common.LogedRecover},
	{"ranged", common.RangedArr, common.RangedRecover},
}
var scaleFunc = []struct {
	scale    string
	transfer func(src []float64) ([]float64, float64, float64)
	reverse  func(src []float64, minNum, maxNum float64) []float64
}{
	{"null", nullScaleFunc, nullScaleRecoverFunc},
	{"zscore", common.ZScoreNormArr, common.ZScoreNormRecover},
	{"minmax", common.MinMaxRangedArr, common.MinMaxRangedRecover},
}
var delFunc = []struct {
	del      string
	transfer func(src []float64) []float64
	reverse  func(src []float64) []float64
}{
	{"null", nullFunc, nullFunc},
	{"delta", common.DeltaArr, common.DeltaRecover},
	{"deltaOfdelta", common.DeltaOfDeltaArr, common.DeltaOfDeltaRecover},
}
var compressFunc = []struct {
	algo       string
	compress   func(dst []byte, src []float64) []byte
	decompress func(dst []float64, src []byte) ([]float64, error)
}{
	{"huffman", huffman.CompressFloat, huffman.DecompressFloat},
	{"elf", elf.CompressFloat, elf.DecompressFloat},
	{"chimp128", chimp128.CompressFloat, chimp128.DecompressFloat},
	{"fpc", fpc.CompressFloat, fpc.DecompressFloat},
	{"zstd", zstd.CompressFloat, zstd.DecompressFloat},
}

func newCSVWriter(path string) *csv.Writer {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	return csv.NewWriter(f)
}
func flattenStats(s *common.TimeSeriesStats) []float64 {
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
func nullScaleFunc(src []float64) ([]float64, float64, float64) {
	return src, 0, 0
}
func nullScaleRecoverFunc(src []float64, minNum, maxNum float64) []float64 {
	return src
}
