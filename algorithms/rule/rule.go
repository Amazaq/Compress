package rule

import (
	"fmt"
	"myalgo/common"
)

const ruleDebug = true

func CompressFloat(dst []byte, src []float64) []byte {
	stats := common.AnalyzeTimeSeries(src)
	// print stats info
	if ruleDebug {
		fmt.Println(stats)
	}
	if stats.UniqueRatio < 0.1 {
		//huffman
	} else if stats.RunLength.InRunRatio > 0.85 {
		// RLE
	} else {
		//zstd
	}
	return nil
}
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	return nil, nil
}
