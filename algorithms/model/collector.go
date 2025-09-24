package model

import (
	"fmt"
	"log"
	"math"
	"myalgo/algorithms/rule"
	"myalgo/common"
	"strconv"
)

const segmentLen = 5000 // 每段长度
var trainDataLen = 0    // 训练集大小

func GetTrainData() {
	featureFile := "./dataset/train_features.csv"
	labelFile := "./dataset/train_labels.csv"
	featWriter := newCSVWriter(featureFile)
	labelWriter := newCSVWriter(labelFile)

	numbers, err := common.GetBigData()
	fmt.Println("numbers length:", len(numbers))
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
			strconv.Itoa(param[3]),
		})
		fmt.Printf("Generate train data: %d / %d \n", i/segmentLen, trainDataLen)
	}

	featWriter.Flush()
	labelWriter.Flush()

	fmt.Println("✅ 已生成训练集:", featureFile, labelFile)
}
func findBestCombo(segment []float64) []int {
	bestSize := math.Inf(1)
	param := make([]int, 4)
	for ranged := range rangedFunc {
		rangedData, _ := rangedFunc[ranged].transfer(segment)
		for scale := range scaleFunc {
			scaledData, _, _ := scaleFunc[scale].transfer(rangedData)
			for del := range delFunc {
				delData := delFunc[del].transfer(scaledData)
				for algo := range compressFunc {
					var compressed []byte
					compressed = compressFunc[algo].compress(compressed, delData)
					size := float64(len(compressed))
					if size < bestSize {
						bestSize = size
						param[0], param[1], param[2], param[3] = ranged, scale, del, algo
					}
				}
			}
		}
	}
	return param
}
