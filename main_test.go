package main

import (
	"fmt"
	"math"
	"math/rand"
	"myalgo/algorithms/chimp"
	"testing"
	"time"
)

var testcases = []struct {
	algo       string
	compress   func([]byte, []uint64) []byte
	decompress func([]uint64, []byte) ([]uint64, error)
}{

	{"chimp", chimp.Compress, chimp.Decompress},
	// {"myalgo", myal.Compress, myal.Decompress},
	// {"fpc", fpc.Compress, fpc.Decompress},
	// {"gorillaz", gorillaz.Compress, gorillaz.Decompress},
	// {"lz4", lz4.Compress, lz4.Decompress},
	// {"lzw", lzw.Compress, lzw.Decompress},
	// {"snappy", snappy.Compress, snappy.Decompress},
	// {"tsxor", tsxor.Compress, tsxor.Decompress},
}

func TestMockedFloats(t *testing.T) {
	for _, tcase := range testcases {
		fmt.Printf("%s ", tcase.algo)
		t.Run(tcase.algo, func(t *testing.T) {
			testMockedFloats(t, tcase.compress, tcase.decompress)
		})
	}
}

func TestRandFloats(t *testing.T) {
	for _, tcase := range testcases {
		fmt.Printf("%s ", tcase.algo)
		t.Run(tcase.algo, func(t *testing.T) {
			testRandFloats(t, tcase.compress, tcase.decompress)
		})
	}
}

func testMockedFloats(t *testing.T, compress func([]byte, []uint64) []byte, decompress func([]uint64, []byte) ([]uint64, error)) {
	t.Helper()
	int64s := []uint64{11123, 2123123, 12312313}
	// 计算压缩前数据大小
	uncompressedSize := len(int64s) * 8 // 每个 float64 占用 8 个字节
	fmt.Printf("压缩前字节数:%d ", uncompressedSize)
	var compressedByte []byte
	compressedByte = compress(compressedByte, int64s)

	compressedSize := len(compressedByte)
	fmt.Printf("压缩后字节数:%d ", compressedSize)
	// 计算压缩比
	compressionRatio := float64(uncompressedSize) / float64(compressedSize)
	// 输出压缩比
	fmt.Printf("Compression ratio: %.2f\n", compressionRatio)

	var decompressedInt64s []uint64
	decompressedInt64s, err := decompress(decompressedInt64s, compressedByte)
	if err != nil {
		t.Error(err)
	}
	if len(decompressedInt64s) != len(int64s) {
		t.Error("de-compress error")
	}
	for i := 0; i < len(decompressedInt64s); i++ {
		if decompressedInt64s[i] != int64s[i] {
			t.Error("de-compress error")
		}
	}
}

func testRandFloats(t *testing.T, compress func([]byte, []uint64) []byte, decompress func([]uint64, []byte) ([]uint64, error)) {
	t.Helper()
	t.Helper()
	var float64s []uint64
	rand.Seed(114514)
	for i := 0; i < 8000; i++ {
		float64s = append(float64s, math.Float64bits(rand.Float64()))
	}
	var compressedByte []byte
	start := time.Now()
	compressedByte = compress(compressedByte, float64s)
	end := time.Now()
	fmt.Print("Compress time:", end.Sub(start))
	var decompressedInt64s []uint64
	startde := time.Now()
	decompressedInt64s, err := decompress(decompressedInt64s, compressedByte)
	endde := time.Now()
	fmt.Println(" | Decompress time:", endde.Sub(startde))
	if err != nil {
		t.Error(err)
	}
	if len(decompressedInt64s) != len(float64s) {
		t.Error("de-compress error")
	}
	for i := 0; i < len(decompressedInt64s); i++ {
		if decompressedInt64s[i] != float64s[i] {
			t.Errorf("de-compress error %d, want %d get %d", i, float64s[i], decompressedInt64s[i])
		}
	}
}
