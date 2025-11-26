package gorillaz

import (
	"myalgo/common"
	"testing"
)

func TestXorWithAndWithoutErase(t *testing.T) {
	// 测试数据
	data, _ := common.ReadDataFromFile("../../dataset/test/Stocks-DE.csv", 10000, 0, 0)

	err := XorWithAndWithoutErase(data)
	if err != nil {
		t.Fatalf("XorWithAndWithoutErase failed: %v", err)
	}

	t.Log("XorWithAndWithoutErase completed successfully")
	t.Log("Results written to dataset/add.txt")
}

func TestXorWithAndWithoutErase_LargeDataset(t *testing.T) {
	// 生成测试数据
	data := make([]float64, 1000)
	base := 20.0
	for i := range data {
		// 模拟温度数据，有小的波动
		data[i] = base + float64(i)*0.01
	}

	err := XorWithAndWithoutErase(data)
	if err != nil {
		t.Fatalf("XorWithAndWithoutErase failed: %v", err)
	}

	t.Logf("Processed %d values successfully", len(data))
}

func TestXorWithAndWithoutErase_SpecialValues(t *testing.T) {
	// 测试特殊值
	data := []float64{
		0.0,
		1.0,
		-1.0,
		100.0,
		0.001,
		1e10,
		1e-10,
	}

	err := XorWithAndWithoutErase(data)
	if err != nil {
		t.Fatalf("XorWithAndWithoutErase failed: %v", err)
	}

	t.Log("Special values test completed successfully")
}
