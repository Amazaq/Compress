package model

import (
	"encoding/json"
	"fmt"
	"log"
	"myalgo/algorithms/rule"
	"os"
	"os/exec"
)

func CompressFloat(dst []byte, src []float64) []byte {
	// 得到数据特征
	features := rule.AnalyzeTimeSeries(src)
	file, _ := os.Create("features.json")
	json.NewEncoder(file).Encode(features)
	file.Close()
	// run python code
	cmd := exec.Command("python", "infer.py", "features.json")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	var result []int
	json.Unmarshal(out, &result)
	dst = RunCompressWithParam(dst, src, result)
	return dst
}

func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	dst, err := RunDecompress(dst, src)
	if err != nil {
		fmt.Println(err)
	}
	return dst, err
}
