package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"myalgo/algorithms/brotli"
	"myalgo/algorithms/lz4"
	"myalgo/algorithms/numerical"
	"myalgo/algorithms/simple8b"
	"myalgo/algorithms/snappy"
	"myalgo/algorithms/zstd"
	"myalgo/common"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const datasetPath = "./dataset"
const resultPath = datasetPath + "/result.csv"

var filenames = []string{
	//22 datasets
	"/Air-pressure.csv",
	"/Air-sensor.csv",
	"/Basel-temp.csv",
	"/Basel-wind.csv",
	"/Bird-migration.csv",
	"/Bitcoin-price.csv",
	"/Blockchain-tr.csv",
	"/City-temp.csv",
	"/City-lat.csv",
	"/City-lon.csv",
	"/Dew-point-temp.csv",
	"/electric_vehicle_charging.csv",
	"/Food-price.csv",
	"/IR-bio-temp.csv",
	"/PM10-dust.csv",
	"/SSD-bench.csv",
	"/POI-lat.csv",
	"/POI-lon.csv",
	"/Stocks-DE.csv",
	"/Stocks-UK.csv",
	"/Stocks-USA.csv",
	"/Wind-Speed.csv",
}
var testcase = []struct {
	algo            string
	CompressFloat   func([]byte, []float64) []byte
	DecompressFloat func([]float64, []byte) ([]float64, error)
}{
	{"zstd", zstd.CompressFloat, zstd.DecompressFloat},
	{"lz4", lz4.CompressFloat, lz4.DecompressFloat},
	{"snappy", snappy.CompressFloat, snappy.DecompressFloat},
	{"brotli", brotli.CompressFloat, brotli.DecompressFloat},
	{"numerical", numerical.CompressFloat, numerical.DecompressFloat},
	// {"huffman", huffman.CompressFloat, huffman.DecompressFloat},
	// {"elf", elf.CompressFloat, elf.DecompressFloat},
	// {"chimp128", chimp128.CompressFloat, chimp128.DecompressFloat},
	// {"chimp", chimp.CompressFloat, chimp.DecompressFloat},
	// {"gorilla", gorillaz.CompressFloat, gorillaz.DecompressFloat},
	// {"gorillaSub", gorillaz.CompressFloatSub, gorillaz.DecompressFloatSub},
	// {"fpc", fpc.CompressFloat, fpc.DecompressFloat},
	// {"model", model.CompressFloat, model.DecompressFloat},
	// {"rule", rule.CompressFloat, rule.DecompressFloat},
	// {"xor", xor.CompressFloat, xor.DecompressFloat},
}

// 将浮点数及其二进制表示写入文件的函数
func WriteFloatBinaryToFile(t *testing.T, float64s []float64, m int) {
	// 如果m大于数组长度，则使用数组长度
	if m > len(float64s) {
		m = len(float64s)
	}

	// 确保dataInfo目录存在
	dataDir := filepath.Join("dataInfo")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, 0755)
	}

	// 创建并写入data.txt文件
	dataFilePath := filepath.Join(dataDir, "data.txt")
	dataFile, err := os.Create(dataFilePath)
	if err != nil {
		t.Fatalf("无法创建数据文件: %v", err)
	}
	defer dataFile.Close()

	// 写入浮点数及其二进制表示
	for i := 0; i < m; i++ {
		f := float64s[i]
		// 获取浮点数的二进制表示
		bits := math.Float64bits(f)
		binaryStr := fmt.Sprintf("%064b", bits) // 确保64位对齐

		// 写入文件
		_, err := fmt.Fprintf(dataFile, "%.10f\t%s\n", f, binaryStr)
		if err != nil {
			t.Fatalf("写入数据文件失败: %v", err)
		}
	}

	fmt.Printf("已将%d个浮点数及其二进制表示写入 %s\n", m, dataFilePath)
}
func TestCompressor(t *testing.T) {
	testLen := len(testcase)
	resultArray := make([][]float64, testLen)
	for i := range resultArray {
		resultArray[i] = make([]float64, 3)
	}
	for index, tcase := range testcase {
		fmt.Printf("%s ", tcase.algo)
		t.Run(tcase.algo, func(t *testing.T) {
			testCompressor(t, resultArray[index], tcase.CompressFloat, tcase.DecompressFloat)
		})
	}
	resultWriter := newCSVWriter(resultPath)
	defer resultWriter.Flush()
	for index, tcase := range testcase {
		resultWriter.Write([]string{
			tcase.algo, // 算法名称
			strconv.FormatFloat(resultArray[index][0], 'f', 6, 64), // 平均压缩比
			strconv.FormatFloat(resultArray[index][1], 'f', 0, 64), // 平均压缩时间(纳秒)
			strconv.FormatFloat(resultArray[index][2], 'f', 0, 64), // 平均解压时间(纳秒)
		})
	}
}
func testCompressor(t *testing.T, result []float64, CompressFloat func([]byte, []float64) []byte, DecompressFloat func([]float64, []byte) ([]float64, error)) {
	n := 100000
	resultWriter := newCSVWriter(resultPath)
	totalRatio := 0.0
	totalCompressTime := 0.0
	totalDecompressTime := 0.0

	// 定义测试数据文件夹
	// testFolder := datasetPath + "/ucrtest"
	testFolder := datasetPath + "/test"

	// 读取文件夹下的所有CSV文件
	entries, err := os.ReadDir(testFolder)
	if err != nil {
		t.Fatalf("无法读取测试文件夹 %s: %v", testFolder, err)
	}

	var testFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".csv" {
			testFiles = append(testFiles, entry.Name())
		}
	}
	if len(testFiles) == 0 {
		t.Fatalf("测试文件夹 %s 中没有找到CSV文件", testFolder)
	}

	for _, file := range testFiles {
		fmt.Printf("/%s ", file)
		filepath := filepath.Join(testFolder, file)
		float64s, _ := common.ReadDataFromFile(filepath, n, 0, 0)
		var compressedByte []byte
		start := time.Now()
		compressedByte = CompressFloat(compressedByte, float64s)
		end := time.Now()
		fmt.Printf("Original size: %d |", len(float64s)*8)
		fmt.Printf("Compressed size: %d |", len(compressedByte))
		ratio := float64(len(float64s)*8) / float64(len(compressedByte))
		totalRatio += ratio
		totalCompressTime += float64(end.Sub(start))
		fmt.Printf("Compression Ratio: %f\n", ratio)
		fmt.Print("Compress time:", end.Sub(start))
		startde := time.Now()
		var decompressedFloat64s []float64
		decompressedFloat64s, err := DecompressFloat(decompressedFloat64s, compressedByte)
		endde := time.Now()
		fmt.Println(" | Decompress time:", endde.Sub(startde))

		if err != nil {
			t.Error(err)
		}
		if len(decompressedFloat64s) != len(float64s) {
			fmt.Printf("first value: %f\n", decompressedFloat64s[0])
			fmt.Printf("Decompressed data length:%d\n", len(decompressedFloat64s))
			t.Error("de-compress error")
		}
		badRecover := 0
		for i := 0; i < len(decompressedFloat64s); i++ {
			if decompressedFloat64s[i] != float64s[i] {
				badRecover++
				t.Errorf("de-compress error %v, want %f get %v", i, float64s[i], decompressedFloat64s[i])
				if badRecover == 10 {
					break
				}
			}
		}
		fmt.Println("-----------------------------------------")
		totalDecompressTime += float64(endde.Sub(startde))
		resultWriter.Write([]string{
			strconv.FormatFloat(ratio, 'f', -1, 64),
			end.Sub(start).String(),
			endde.Sub(startde).String(),
		})
	}
	filenlen := len(testFiles)
	result[0] = totalRatio / float64(filenlen)
	result[1] = totalCompressTime / float64(filenlen)
	result[2] = totalDecompressTime / float64(filenlen)
	// resultWriter.Write([]string{
	// 	strconv.FormatFloat(totalRatio, 'f', -1, 64),
	// 	strconv.FormatFloat(totalCompressTime, 'f', -1, 64),
	// 	strconv.FormatFloat(totalDecompressTime, 'f', -1, 64),
	// })
	resultWriter.Flush()
}
func TestFloats(t *testing.T) {
	n := 100
	// m := 100
	// float64s, _ := ReadDataFromFile("./dataset/city_temperature.csv", n, 0, 2)
	float64s, _ := common.ReadDataFromFile("./dataset/test/Stocks-DE.csv", n, 0, 0)
	// float64s, _ := common.ReadDataFromFile("./dataset/test/Air-sensor.csv", n, 0, 0)
	// float64s = common.DeltaArr((float64s))
	// float64s, _ := ReadDataFromFile("./dataset/air-sensor.csv", n, 4, 4)
	// float64s, _ := ReadDataFromFile("./dataset/temperature_wind.csv", n, 10, 1)
	// WriteFloatBinaryToFile(t, float64s, m)

	for _, tcase := range testcase {
		fmt.Printf("%s ", tcase.algo)
		t.Run(tcase.algo, func(t *testing.T) {
			testCSVFloats(t, float64s, tcase.CompressFloat, tcase.DecompressFloat)
		})
	}
}
func testCSVFloats(t *testing.T, float64s []float64, CompressFloat func([]byte, []float64) []byte, DecompressFloat func([]float64, []byte) ([]float64, error)) {

	var compressedByte []byte
	start := time.Now()
	compressedByte = CompressFloat(compressedByte, float64s)
	end := time.Now()
	fmt.Printf("Original size: %d |", len(float64s)*8)
	fmt.Printf("Compressed size: %d |", len(compressedByte))
	ratio := float64(len(float64s)*8) / float64(len(compressedByte))
	fmt.Printf("Compression Ration: %f\n", ratio)
	fmt.Print("Compress time:", end.Sub(start))
	startde := time.Now()
	var decompressedFloat64s []float64
	decompressedFloat64s, err := DecompressFloat(decompressedFloat64s, compressedByte)
	endde := time.Now()
	fmt.Println(" | Decompress time:", endde.Sub(startde))
	fmt.Println("-----------------------------------------")
	if err != nil {
		t.Error(err)
	}
	if len(decompressedFloat64s) != len(float64s) {
		fmt.Printf("first value: %f\n", decompressedFloat64s[0])
		fmt.Printf("Decompressed data length:%d\n", len(decompressedFloat64s))
		t.Error("de-compress error")
	}
	for i := 0; i < len(decompressedFloat64s); i++ {
		if decompressedFloat64s[i] != float64s[i] {
			t.Errorf("de-compress error %v, want %f get %v", i, float64s[i], decompressedFloat64s[i])
		}
	}
}
func TestReadData(t *testing.T) {
	t.Helper()
	n := 10
	data, err := common.ReadDataFromFile("./dataset/city_temperature.csv", n, 1, 3)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
	fmt.Printf("%d", len(data))
}

func TestFloatSplit(t *testing.T) {
	n := 20
	float64s, _ := common.ReadDataFromFile("./dataset/city_temperature.csv", n, 1, 3)
	for _, f := range float64s {
		intPart, fracPart := math.Modf(f)
		fracPart *= 10000000
		fracPart /= 10000000
		fmt.Printf("int:%d, frac: %f | num: %f , add:%f|\n", int(intPart), fracPart, f, intPart+fracPart)
	}
}

func TestModf(t *testing.T) {
	val := 64.2
	intPart, fracPart := math.Modf(val)
	fmt.Printf("%f, %f", intPart, fracPart)
}

func TestSimple8bDebug(t *testing.T) {
	// 创建原始数组
	arr := make([]uint64, 250)

	// 验证初始状态
	fmt.Println("=== 验证初始数组状态 ===")
	for i := 0; i < 10; i++ {
		if arr[i] != 0 {
			t.Fatalf("Initial array not zero at index %d: %d", i, arr[i])
		}
	}
	fmt.Printf("前10个元素都是0, 数组长度: %d\n", len(arr))

	// 保存原始数组的副本用于对比
	originalArr := make([]uint64, len(arr))
	copy(originalArr, arr)

	fmt.Println("\n=== 压缩过程 ===")
	var compressBytes []byte
	compressBytes = simple8b.Compress(compressBytes, arr)
	fmt.Printf("压缩后字节数: %d\n", len(compressBytes))

	// 检查压缩后原始数组是否被修改
	fmt.Println("\n=== 检查原始数组是否被压缩过程修改 ===")
	arrayModified := false
	for i := 0; i < len(arr); i++ {
		if arr[i] != originalArr[i] {
			fmt.Printf("原始数组被修改! 位置 %d: 原来=%d, 现在=%d\n", i, originalArr[i], arr[i])
			arrayModified = true
			if i > 50 { // 只显示前50个错误
				fmt.Println("... (更多错误)")
				break
			}
		}
	}
	if !arrayModified {
		fmt.Println("原始数组未被修改")
	}

	fmt.Println("\n=== 解压过程 ===")
	decompressBuffer := make([]uint64, len(arr))
	recovered, err := simple8b.Decompress(decompressBuffer, compressBytes)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("解压后长度: %d\n", len(recovered))

	// 对比解压结果和原始数组（使用副本）
	fmt.Println("\n=== 验证解压结果 ===")
	mismatchCount := 0
	for i := 0; i < len(recovered) && i < len(originalArr); i++ {
		if recovered[i] != originalArr[i] {
			if mismatchCount < 10 { // 只显示前10个错误
				fmt.Printf("不匹配位置 %d: 原始=%d, 解压=%d\n", i, originalArr[i], recovered[i])
			}
			mismatchCount++
		}
		if recovered[i] != 0 {
			fmt.Printf("error %d %d", i, recovered[i])
		}
	}

	if mismatchCount == 0 {
		fmt.Println("✓ 所有值都匹配!")
	} else {
		fmt.Printf("✗ 发现 %d 个不匹配的值\n", mismatchCount)
	}
}
func newCSVWriter(path string) *csv.Writer {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return csv.NewWriter(f)
}
