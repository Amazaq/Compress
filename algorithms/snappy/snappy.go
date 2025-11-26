package snappy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"

	"github.com/klauspost/compress/snappy"
)

func Compress(dst []byte, src []uint64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], u)
	}
	bw := &bytes.Buffer{}
	snappyEncoder := snappy.NewBufferedWriter(bw)
	_, err := snappyEncoder.Write(uncb)
	if err != nil {
		fmt.Println(err)
	}
	err = snappyEncoder.Flush()
	if err != nil {
		fmt.Println(err)
	}
	err = snappyEncoder.Close()
	if err != nil {
		fmt.Println(err)
	}
	return append(dst, bw.Bytes()...)
}
func CompressFloat(dst []byte, src []float64) []byte {
	uncb := make([]byte, len(src)*8)
	for i, u := range src {
		binary.LittleEndian.PutUint64(uncb[i*8:(i+1)*8], math.Float64bits(u))
	}
	bw := &bytes.Buffer{}
	snappyEncoder := snappy.NewBufferedWriter(bw)
	_, err := snappyEncoder.Write(uncb)
	if err != nil {
		fmt.Println(err)
	}
	err = snappyEncoder.Flush()
	if err != nil {
		fmt.Println(err)
	}
	err = snappyEncoder.Close()
	if err != nil {
		fmt.Println(err)
	}
	return append(dst, bw.Bytes()...)
}
func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	snappyDecoder := snappy.NewReader(bytes.NewReader(src))
	uncb, err := ioutil.ReadAll(snappyDecoder)
	if err != nil {
		fmt.Println(err)
	}
	for i := 0; i < len(uncb)/8; i++ {
		dst = append(dst, binary.LittleEndian.Uint64(uncb[i*8:(i+1)*8]))
	}
	return dst, nil
}
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	snappyDecoder := snappy.NewReader(bytes.NewReader(src))
	uncb, err := ioutil.ReadAll(snappyDecoder)
	if err != nil {
		fmt.Println(err)
	}
	for i := 0; i < len(uncb)/8; i++ {
		dst = append(dst, math.Float64frombits(binary.LittleEndian.Uint64(uncb[i*8:(i+1)*8])))
	}
	return dst, nil
}
