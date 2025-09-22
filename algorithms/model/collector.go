package model

import (
	"fmt"
	"log"
	"math"
	"myalgo/algorithms/chimp"
	"myalgo/algorithms/huffman"
	"myalgo/algorithms/rule"
	"myalgo/common"
	"strconv"
)

const segmentLen = 5000 // 每段长度
var trainDataLen = 750  // 训练集大小
var delFunc = []struct {
	del      string
	transfer func(src []float64) []float64
	reverse  func(src []float64) []float64
}{
	{"delta", rule.DeltaArr, rule.DeltaRecover},
	{"deltaOfdelta", rule.DeltaOfDeltaArr, rule.DeltaOfDeltaRecover},
	{"null", nullFunc, nullFunc},
}
var rangedFunc = []struct {
	ranged   string
	transfer func(src []float64) ([]float64, float64)
	reverse  func(src []float64, base float64) []float64
}{
	{"ranged", rule.RangedArr, rule.RangedRecover},
	{"null", nullRangedFunc, nullRangedRecoverFunc},
}
var compressFunc = []struct {
	algo       string
	compress   func(dst []byte, src []float64) []byte
	decompress func(dst []float64, src []byte) ([]float64, error)
}{
	{"huffman", huffman.CompressFloat, huffman.DecompressFloat},
	{"chimp", chimp.CompressFloat, chimp.DecompressFloat},
}

func GetTrainData(datafile string) {
	featureFile := "./dataset/train_features.csv"
	labelFile := "./dataset/train_labels.csv"
	featWriter := newCSVWriter(featureFile)
	labelWriter := newCSVWriter(labelFile)

	numbers, err := common.GetBigData()
	trainDataLen = len(numbers) / segmentLen
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i+segmentLen <= len(numbers); i += segmentLen {
		segment := numbers[i : i+segmentLen]

		stats := rule.AnalyzeTimeSeries((segment))
		features := flattenStats(stats)

		// 写特征
		featRow := make([]string, len(features))
		for j, f := range features {
			featRow[j] = fmt.Sprintf("%.6f", f)
		}
		featWriter.Write(featRow)

		// 遍历所有压缩方法
		param := findBestCombo(segment)

		labelWriter.Write([]string{
			strconv.Itoa(param[0]),
			strconv.Itoa(param[1]),
			strconv.Itoa(param[2]),
		})
		fmt.Printf("Generate train data: %d / %d \n", i/segmentLen, trainDataLen)
	}

	featWriter.Flush()
	labelWriter.Flush()

	fmt.Println("✅ 已生成训练集:", featureFile, labelFile)
}
func findBestCombo(segment []float64) []int {
	bestSize := math.Inf(1)
	param := make([]int, 3)
	for diff := range delFunc {
		diffed := delFunc[diff].transfer(segment)

		for sub := range rangedFunc {
			subbed, _ := rangedFunc[sub].transfer(diffed)

			for algo := range compressFunc {
				var compressed []byte
				compressed = compressFunc[algo].compress(compressed, subbed)
				size := float64(len(compressed))
				if size < bestSize {
					bestSize = size
					param[0], param[1], param[2] = diff, sub, algo
				}
			}
		}
	}
	return param
}
