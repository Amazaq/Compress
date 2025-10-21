package model

import (
	"fmt"
	"log"
	"math"
	"myalgo/common"
	"strconv"
)

const segmentLen = 5000 // 每段长度
var trainDataLen = 0    // 训练集大小

func GetTrainData(dataset int) {
	featureFile := "./dataset/train_features.csv"
	labelFile := "./dataset/train_labels.csv"
	featWriter := newCSVWriter(featureFile)
	labelWriter := newCSVWriter(labelFile)
	// 把两种情况统一起来 读取的数据都用numbers表示
	var numbers []float64
	var err error
	if dataset == 1 {
		numbers, err = common.ReadDataFromFile("./dataset/train/ucr.csv", 80110000, 0, 0)
		if err != nil {
			fmt.Println("Read Data Error!!!")
		}
	} else {
		numbers, err = common.GetBigData()
	}
	fmt.Println("numbers length:", len(numbers))
	trainDataLen = len(numbers) / segmentLen
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i+segmentLen <= len(numbers); i += segmentLen {
		segment := numbers[i : i+segmentLen]

		stats := common.AnalyzeTimeSeries((segment))
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
	bestParam := make([]int, 4)
	for ranged := range rangedFunc {
		for scale := range scaleFunc {
			for del := range delFunc {
				for algo := range compressFunc {
					var dst []byte
					param[0] = ranged
					param[1] = scale
					param[2] = del
					param[3] = algo
					dst = RunCompressWithParam(dst, segment, param)
					size := float64(len(dst))
					if size < bestSize {
						bestSize = size
						bestParam[0], bestParam[1], bestParam[2], bestParam[3] = ranged, scale, del, algo
					}
				}
			}
		}
	}
	return bestParam
}
