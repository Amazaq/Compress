package common

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func ReadDataFromFile(filePath string, limit int, skip int, column int) ([]float64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file error '%s': %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	//跳过标头
	for i := 0; i < skip; i++ {
		_, _ = reader.Read()
	}
	data := make([]float64, 0, limit)

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read data error: %w", err)
		}
		if len(record) <= column {
			continue
		}

		tempStr := record[column]
		temp, err := strconv.ParseFloat(tempStr, 64)
		if err != nil {
			continue
		}
		data = append(data, temp)

		if len(data) >= limit {
			break
		}
	}

	return data, nil
}

// ReadDataFromFileWithStrings 读取数据并同时返回字符串数组（用于精度检测）
func ReadDataFromFileWithStrings(filePath string, limit int, skip int, column int) ([]float64, []string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open file error '%s': %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	//跳过标头
	for i := 0; i < skip; i++ {
		_, _ = reader.Read()
	}
	data := make([]float64, 0, limit)
	dataStrings := make([]string, 0, limit)

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("read data error: %w", err)
		}
		if len(record) <= column {
			continue
		}

		tempStr := record[column]
		temp, err := strconv.ParseFloat(tempStr, 64)
		if err != nil {
			continue
		}
		data = append(data, temp)
		dataStrings = append(dataStrings, tempStr)

		if len(data) >= limit {
			break
		}
	}

	return data, dataStrings, nil
}

func GetBigData() ([]float64, error) {
	var arr []float64

	fileData, err := ReadDataFromFile("./dataset/train/air_sensor.csv", 10000, 1, 4)
	if err != nil {
		fmt.Println("Read Data Error!!!")
		return nil, err
	}
	arr = append(arr, fileData...)

	fileData, err = ReadDataFromFile("./dataset/train/city_temp.csv", 2900000, 1, 7)
	if err != nil {
		fmt.Println("Read Data Error!!!")
		return nil, err
	}
	arr = append(arr, fileData...)

	fileData, err = ReadDataFromFile("./dataset/train/migration_original.csv", 89000, 1, 3)
	if err != nil {
		fmt.Println("Read Data Error!!!")
		return nil, err
	}
	arr = append(arr, fileData...)

	fileData, err = ReadDataFromFile("./dataset/train/wind_temp.csv", 750000, 10, 1)
	if err != nil {
		fmt.Println("Read Data Error!!!")
		return nil, err
	}
	arr = append(arr, fileData...)
	fileData, err = ReadDataFromFile("./dataset/train/wind_temp.csv", 750000, 10, 2)
	if err != nil {
		fmt.Println("Read Data Error!!!")
		return nil, err
	}
	arr = append(arr, fileData...)
	fileData, err = ReadDataFromFile("./dataset/train/wind_temp.csv", 750000, 10, 3)
	if err != nil {
		fmt.Println("Read Data Error!!!")
		return nil, err
	}
	arr = append(arr, fileData...)

	return arr, nil
}

// MergeCSVsByPattern 根据文件名模式合并 CSV 文件
// pattern: 文件名中需要包含的字符串，如 "_1_minute", "_2min", "_30min" 等
func MergeCSVsByPattern(srcDir, outFile, pattern string) error {
	out, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("create output file error: %w", err)
	}
	defer out.Close()

	writer := csv.NewWriter(out)
	defer writer.Flush()

	firstHeader := true
	fileCount := 0

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 只处理文件且文件名包含指定模式的 CSV 文件
		if !info.IsDir() && strings.Contains(info.Name(), pattern) && strings.HasSuffix(info.Name(), ".csv") {
			fmt.Printf("Processing: %s\n", path)
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("open file error '%s': %w", path, err)
			}
			defer f.Close()

			r := csv.NewReader(f)
			lineNum := 0
			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("read csv error '%s': %w", path, err)
				}

				// 只写第一个文件的表头
				if lineNum == 0 {
					if firstHeader {
						if err := writer.Write(record); err != nil {
							return fmt.Errorf("write header error: %w", err)
						}
						firstHeader = false
					}
				} else {
					if err := writer.Write(record); err != nil {
						return fmt.Errorf("write record error: %w", err)
					}
				}
				lineNum++
			}
			fileCount++
		}
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("合并完成！共处理 %d 个文件\n", fileCount)
	return nil
}

// ExtractUCRDatasetValues 提取 UCR 数据集中所有数值（去掉类别标签）
func ExtractUCRDatasetValues(baseDir string) error {
	// 遍历 UCRArchive_2018 下的所有子文件夹
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("无法读取目录 %s: %v", baseDir, err)
	}

	processedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		datasetName := entry.Name()
		datasetPath := filepath.Join(baseDir, datasetName)

		// 查找 TRAIN 和 TEST 文件
		trainFile := filepath.Join(datasetPath, datasetName+"_TRAIN.tsv")
		testFile := filepath.Join(datasetPath, datasetName+"_TEST.tsv")

		// 检查文件是否存在
		trainExists := fileExists(trainFile)
		testExists := fileExists(testFile)

		if !trainExists && !testExists {
			fmt.Printf("跳过 %s: 未找到 TRAIN 或 TEST 文件\n", datasetName)
			continue
		}

		// 提取数值并写入新文件
		outputFile := filepath.Join(datasetPath, "all_values.txt")
		err := extractAndMergeValues(trainFile, testFile, outputFile, trainExists, testExists)
		if err != nil {
			fmt.Printf("处理 %s 时出错: %v\n", datasetName, err)
			continue
		}

		processedCount++
		fmt.Printf("✓ 已处理 %s -> %s\n", datasetName, outputFile)
	}

	fmt.Printf("\n总共处理了 %d 个数据集\n", processedCount)
	return nil
}

// extractAndMergeValues 从 TRAIN 和 TEST 文件中提取数值并合并到一个文件
func extractAndMergeValues(trainFile, testFile, outputFile string, trainExists, testExists bool) error {
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("无法创建输出文件: %v", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// 处理 TRAIN 文件
	if trainExists {
		err = processFile(trainFile, writer)
		if err != nil {
			return fmt.Errorf("处理 TRAIN 文件失败: %v", err)
		}
	}

	// 处理 TEST 文件
	if testExists {
		err = processFile(testFile, writer)
		if err != nil {
			return fmt.Errorf("处理 TEST 文件失败: %v", err)
		}
	}

	return nil
}

// processFile 处理单个文件，提取所有数值（去掉第一列的类别标签）
func processFile(filename string, writer *bufio.Writer) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 增加缓冲区大小以处理长行
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 按制表符分割
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			// 尝试空格分割
			fields = strings.Fields(line)
			if len(fields) < 2 {
				fmt.Printf("  警告: %s 第 %d 行格式异常，跳过\n", filename, lineNum)
				continue
			}
		}

		// 跳过第一个字段（类别标签），输出其余所有数值
		for i := 1; i < len(fields); i++ {
			value := strings.TrimSpace(fields[i])
			if value != "" {
				_, err := writer.WriteString(value + "\n")
				if err != nil {
					return err
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取文件时出错: %v", err)
	}

	return nil
}

// fileExists 检查文件是否存在
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// MergeAllValuesToCSV 合并所有 all_values.txt 文件到一个大的 CSV 文件
func MergeAllValuesToCSV(baseDir, outputFile string) error {
	// 遍历 UCRArchive_2018 下的所有子文件夹
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("无法读取目录 %s: %v", baseDir, err)
	}

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("无法创建输出文件: %v", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	totalValues := 0
	datasetCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		datasetName := entry.Name()
		datasetPath := filepath.Join(baseDir, datasetName)
		valuesFile := filepath.Join(datasetPath, "all_values.txt")

		// 检查 all_values.txt 是否存在
		if !fileExists(valuesFile) {
			continue
		}

		// 读取并写入所有数值
		file, err := os.Open(valuesFile)
		if err != nil {
			fmt.Printf("警告: 无法打开 %s: %v\n", valuesFile, err)
			continue
		}

		scanner := bufio.NewScanner(file)
		buf := make([]byte, 0, 1024*1024)
		scanner.Buffer(buf, 10*1024*1024)

		lineCount := 0
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			_, err := writer.WriteString(line + "\n")
			if err != nil {
				file.Close()
				return fmt.Errorf("写入数据时出错: %v", err)
			}
			lineCount++
			totalValues++
		}

		if err := scanner.Err(); err != nil {
			file.Close()
			return fmt.Errorf("读取 %s 时出错: %v", valuesFile, err)
		}

		file.Close()
		datasetCount++
		fmt.Printf("✓ 已合并 %s: %d 个数值\n", datasetName, lineCount)
	}

	fmt.Printf("\n总共合并了 %d 个数据集，共 %d 个数值\n", datasetCount, totalValues)
	return nil
}

// GenerateUCRTestDataset 从UCR数据集随机选择20个子文件夹，提取测试数据
// ucrDir: UCR数据集根目录 (例如: "./dataset/UCRArchive_2018")
// outputDir: 输出测试数据目录 (例如: "./dataset/ucrtest")
// datasetCount: 要选择的数据集数量
// sampleSize: 每个数据集提取的数据量
func GenerateUCRTestDataset(ucrDir, outputDir string, datasetCount, sampleSize int) error {
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 读取所有UCR子文件夹
	entries, err := os.ReadDir(ucrDir)
	if err != nil {
		return fmt.Errorf("读取UCR目录失败: %v", err)
	}

	// 过滤出包含all_values.csv的文件夹
	var validDatasets []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		allValuesPath := filepath.Join(ucrDir, entry.Name(), "all_values.txt")
		if _, err := os.Stat(allValuesPath); err == nil {
			validDatasets = append(validDatasets, entry.Name())
		}
	}

	if len(validDatasets) == 0 {
		return fmt.Errorf("未找到包含 all_values.csv 的数据集")
	}

	fmt.Printf("找到 %d 个有效的UCR数据集\n", len(validDatasets))

	// 随机选择数据集
	if datasetCount > len(validDatasets) {
		datasetCount = len(validDatasets)
		fmt.Printf("⚠️  请求数量超过可用数据集，调整为 %d 个\n", datasetCount)
	}

	// 使用Fisher-Yates洗牌算法随机选择
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	selectedDatasets := make([]string, datasetCount)
	perm := rng.Perm(len(validDatasets))
	for i := 0; i < datasetCount; i++ {
		selectedDatasets[i] = validDatasets[perm[i]]
	}

	fmt.Printf("\n开始生成测试数据集...\n")
	fmt.Printf("每个数据集提取 %d 个连续数值\n", sampleSize)
	fmt.Println(strings.Repeat("=", 80))

	successCount := 0
	for i, datasetName := range selectedDatasets {
		fmt.Printf("\n[%d/%d] 处理 %s\n", i+1, datasetCount, datasetName)

		allValuesPath := filepath.Join(ucrDir, datasetName, "all_values.txt")

		// 读取文件并计算总行数
		file, err := os.Open(allValuesPath)
		if err != nil {
			fmt.Printf("  ✗ 打开文件失败: %v\n", err)
			continue
		}

		// 统计总数据量
		scanner := bufio.NewScanner(file)
		totalCount := 0
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				totalCount++
			}
		}
		file.Close()

		// 读取所有数据到内存
		file, err = os.Open(allValuesPath)
		if err != nil {
			fmt.Printf("  ✗ 重新打开文件失败: %v\n", err)
			continue
		}

		scanner = bufio.NewScanner(file)
		var allData []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// 过滤掉空行和NaN值
			if line != "" && line != "NaN" && line != "nan" && line != "NAN" {
				allData = append(allData, line)
			}
		}
		file.Close()

		if len(allData) == 0 {
			fmt.Printf("  ✗ 没有有效数据（全部为NaN）\n")
			continue
		}

		// 生成数据（有多少用多少）
		var extractedData []string
		if len(allData) >= sampleSize {
			// 数据充足，随机选择起始位置
			maxStartPos := len(allData) - sampleSize
			startPos := rng.Intn(maxStartPos + 1)
			extractedData = allData[startPos : startPos+sampleSize]
			fmt.Printf("  总数据量: %d, 起始位置: %d, 提取: %d 个\n", len(allData), startPos, sampleSize)
		} else {
			// 数据不足，全部使用
			extractedData = allData
			fmt.Printf("  总数据量: %d (不足 %d，使用全部数据)\n", len(allData), sampleSize)
		}

		// 写入输出文件
		outputPath := filepath.Join(outputDir, datasetName+".csv")
		outFile, err := os.Create(outputPath)
		if err != nil {
			fmt.Printf("  ✗ 创建输出文件失败: %v\n", err)
			continue
		}

		writer := bufio.NewWriter(outFile)
		writtenCount := 0
		for _, value := range extractedData {
			// 双重保险：写入前再次检查是否为NaN
			if value != "NaN" && value != "nan" && value != "NAN" {
				_, err := writer.WriteString(value + "\n")
				if err != nil {
					outFile.Close()
					fmt.Printf("  ✗ 写入数据失败: %v\n", err)
					continue
				}
				writtenCount++
			}
		}

		writer.Flush()
		outFile.Close()

		fmt.Printf("  ✓ 成功生成: %s (%d 个数值)\n", outputPath, writtenCount)
		successCount++
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\n✅ 测试数据集生成完成！\n")
	fmt.Printf("成功: %d/%d 个数据集\n", successCount, datasetCount)
	fmt.Printf("输出目录: %s\n", outputDir)

	return nil
}
