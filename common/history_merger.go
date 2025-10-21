package common

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MergeStocksZipFilesByCountry 按国家合并 history 文件夹下 Stocks 类型的 zip 文件
func MergeStocksZipFilesByCountry(historyDir, outputDir string) error {
	// 创建输出文件的 writers
	writers := make(map[string]*csv.Writer)
	files := make(map[string]*os.File)
	firstHeaders := make(map[string]bool)
	fileCounts := make(map[string]int)

	// 支持的国家和对应的文件名
	countries := map[string]string{
		"USA":            "stocks-USA.csv",
		"United Kingdom": "stocks-UK.csv",
		"Germany":        "stocks-DE.csv",
	}

	// 创建输出文件
	for country, filename := range countries {
		outPath := filepath.Join(outputDir, filename)
		f, err := os.Create(outPath)
		if err != nil {
			// 关闭已打开的文件
			for _, openFile := range files {
				openFile.Close()
			}
			return fmt.Errorf("create output file error for %s: %w", country, err)
		}
		files[country] = f
		writers[country] = csv.NewWriter(f)
		firstHeaders[country] = true
		fileCounts[country] = 0
	}

	// 确保所有文件最后都被关闭和刷新
	defer func() {
		for _, w := range writers {
			w.Flush()
		}
		for _, f := range files {
			f.Close()
		}
	}()

	// 获取所有 zip 文件
	zipFiles, err := filepath.Glob(filepath.Join(historyDir, "*.zip"))
	if err != nil {
		return fmt.Errorf("glob zip files error: %w", err)
	}

	fmt.Printf("找到 %d 个 ZIP 文件\n", len(zipFiles))

	for _, zipPath := range zipFiles {
		zipBaseName := filepath.Base(zipPath)

		// 只处理 Stocks 类型的文件
		if !strings.Contains(zipBaseName, "Stocks") {
			continue
		}

		// 判断是哪个国家
		var targetCountry string
		if strings.Contains(zipBaseName, "Stocks USA") {
			targetCountry = "USA"
		} else if strings.Contains(zipBaseName, "Stocks United Kingdom") {
			targetCountry = "United Kingdom"
		} else if strings.Contains(zipBaseName, "Stocks Germany") {
			targetCountry = "Germany"
		} else {
			// 其他国家的 Stocks 文件跳过
			continue
		}

		fmt.Printf("处理 [%s]: %s\n", targetCountry, zipBaseName)

		// 打开 zip 文件
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			fmt.Printf("警告: 无法打开 zip 文件 %s: %v\n", zipPath, err)
			continue
		}

		// 遍历 zip 中的文件
		for _, file := range zipReader.File {
			// 只处理 .his 文件
			if !strings.HasSuffix(strings.ToLower(file.Name), ".his") {
				continue
			}

			rc, err := file.Open()
			if err != nil {
				fmt.Printf("    警告: 无法打开文件 %s: %v\n", file.Name, err)
				continue
			}

			// 读取CSV内容
			r := csv.NewReader(rc)
			writer := writers[targetCountry]
			lineNum := 0

			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Printf("    警告: 读取CSV错误 %s: %v\n", file.Name, err)
					break
				}

				// 只写第一个文件的表头
				if lineNum == 0 {
					if firstHeaders[targetCountry] {
						if err := writer.Write(record); err != nil {
							rc.Close()
							zipReader.Close()
							return fmt.Errorf("write header error: %w", err)
						}
						firstHeaders[targetCountry] = false
					}
				} else {
					if err := writer.Write(record); err != nil {
						rc.Close()
						zipReader.Close()
						return fmt.Errorf("write record error: %w", err)
					}
				}
				lineNum++
			}
			rc.Close()
			fileCounts[targetCountry]++
		}
		zipReader.Close()
	}

	fmt.Println("\n合并完成！统计信息:")
	for country, filename := range countries {
		fmt.Printf("  %s (%s): %d 个文件\n", country, filename, fileCounts[country])
	}
	return nil
}
