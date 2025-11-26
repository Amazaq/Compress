package main

import (
	"myalgo/algorithms/numerical"
	"myalgo/common"
)

func main() {
	// values, strings, _ := common.ReadDataFromFileWithStrings("./dataset/test/POI-lat.csv", 100, 0, 0) //16
	values, strings, _ := common.ReadDataFromFileWithStrings("./dataset/test/Stocks-DE.csv", 100, 0, 0) //3
	// values, strings, _ := common.ReadDataFromFileWithStrings("./dataset/test/SSD-bench.csv", 100, 0, 0) // 1
	// values, strings, _ := common.ReadDataFromFileWithStrings("./dataset/test/Air-pressure.csv", 100, 0, 0) //5
	// values, strings, _ := common.ReadDataFromFileWithStrings("./dataset/test/Basel-temp.csv", 100, 0, 0) //10

	// 检测约束
	nc := numerical.DetectConstraintsWithStrings(values, strings)
	nc.PrintConstraints()

	// 验证数据是否符合约束
	count, anomalies := nc.ValidateConstraints(values, strings)
	numerical.PrintAnomalies(count, anomalies)
}
