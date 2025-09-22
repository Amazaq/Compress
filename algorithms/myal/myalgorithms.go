package myal

import (
	"fmt"
)

type DataType string

const (
	Int64   DataType = "int"
	Float64 DataType = "float"
	Enum    DataType = "enum"
)

type ApplicationSemantics struct {
	//时间序列ID
	TimeSeriesID uint64
	// 数据类型
	DataType DataType
	// 值域(浮点数、整数)
	DataRange []uint64
	// 精度(浮点数)
	Precision uint64
	// 数据直方图
	DataHistogram []uint64
	// 枚举数组(枚举)
	EnumValue []uint64
	// 浮点数压缩算法
	FloatCompressor []string
	// 整数压缩算法
	IntegerCompressor []string
}

func NewApplicationSemantics(dataType DataType, dataRange []uint64, precision uint64, dataHistogram []uint64, enumValue []uint64) *ApplicationSemantics {
	return &ApplicationSemantics{
		DataType:      dataType,
		DataRange:     dataRange,
		Precision:     precision,
		DataHistogram: dataHistogram,
		EnumValue:     enumValue,
	}
}

func NewDefaultApplicationSemantics() *ApplicationSemantics {
	return &ApplicationSemantics{
		DataType:          Float64,
		Precision:         0,
		FloatCompressor:   []string{"chimp"},
		IntegerCompressor: []string{"simple8b"},
	}
}

func PrintDetail(as ApplicationSemantics) {
	fmt.Println("DataType:", as.DataType)
	fmt.Println("DataRange:", as.DataRange)
	fmt.Println("Precision:", as.Precision)
	fmt.Println("DataHist:", as.DataHistogram)
	fmt.Println("EnumValue:", as.EnumValue)
}

func FloatCompress(dst []byte, src []uint64) []byte {
	return dst
}

func IntegerCompress(dst []byte, src []uint64) []byte {
	return dst
}

func Compress(dst []byte, src []uint64) []byte {

	return dst
}

func Decompress(dst []uint64, src []byte) ([]uint64, error) {

	return dst, nil
}
