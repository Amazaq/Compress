package common

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

func Append64(dst []byte, src uint64) []byte {
	dst = append(dst, uint8(src>>56))
	dst = append(dst, uint8(src>>48))
	dst = append(dst, uint8(src>>40))
	dst = append(dst, uint8(src>>32))
	dst = append(dst, uint8(src>>24))
	dst = append(dst, uint8(src>>16))
	dst = append(dst, uint8(src>>8))
	dst = append(dst, uint8(src))
	return dst
}

func Get64(src []byte, i int) (uint64, int, error) {
	if src == nil || len(src[i:]) < 8 {
		return 0, i, io.EOF
	}
	v := uint64(0)
	for j := 0; j < 8; j++ {
		v <<= 8
		v |= uint64(src[i])
		i++
	}
	return v, i - 1, nil
}
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
		if len(record) < 3 {
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

func GetBigData() ([]float64, error) {
	var arr []float64

	fileData, err := ReadDataFromFile("./dataset/train/air_sensor.csv", 10000, 1, 4)
	if err != nil {
		fmt.Println("Read Data Error!!!")
		return nil, err
	}
	arr = append(arr, fileData...)

	fileData, err = ReadDataFromFile("./dataset/train/city_temperature.csv", 2900000, 1, 7)
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
