package model

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
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
	cmd := exec.Command("python", "algorithms/model/infer.py", "features.json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("run python infer code failed: %v\nstdout/stderr: %s", err, string(out))
	}
	var result []int
	if uerr := json.Unmarshal(out, &result); uerr != nil {
		log.Fatalf("invalid python output, expect JSON int array [d,r,c], got: %s, err: %v", string(out), uerr)
	}
	fmt.Println(result)
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
func RunCompressWithParam(dst []byte, src []float64, param []int) []byte {

	copied := make([]float64, len(src))
	copy(copied, src)

	ranged, base := rangedFunc[param[1]].transfer(copied)
	scale, minNum, maxNum := scaleFunc[param[2]].transfer(ranged)
	del := delFunc[param[0]].transfer(scale)

	dst = append(dst, byte(param[0]))
	dst = append(dst, byte(param[1]))
	dst = append(dst, byte(param[2]))
	dst = append(dst, byte(param[3]))
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(base))
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(minNum))
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(maxNum))
	dst = compressFunc[param[2]].compress(dst, del)

	return dst
}
func RunDecompress(dst []float64, src []byte) ([]float64, error) {
	if len(src) < 11 {
		return nil, fmt.Errorf("invalid data: byte slice is too short, received %d bytes, need at least 11", len(src))
	}

	param0 := int(src[0])
	param1 := int(src[1])
	param2 := int(src[2])
	param3 := int(src[3])

	if param0 >= len(delFunc) || param1 >= len(rangedFunc) || param2 >= len(compressFunc) {
		return nil, fmt.Errorf("invalid parameters in data stream: p0=%d, p1=%d, p2=%d", param0, param1, param2)
	}
	offset := 3
	baseBits := binary.LittleEndian.Uint64(src[offset : offset+8])
	base := math.Float64frombits(baseBits)
	offset += 8
	minNumBits := binary.LittleEndian.Uint64(src[offset : offset+8])
	minNum := math.Float64frombits(minNumBits)
	offset += 8
	maxNumBits := binary.LittleEndian.Uint64(src[offset : offset+8])
	maxNum := math.Float64frombits(maxNumBits)
	offset += 8
	decompressor := compressFunc[param2].decompress
	dst, err := decompressor(dst, src[offset:])
	if err != nil {
		return nil, fmt.Errorf("decompression failed using %s: %w", compressFunc[param2].algo, err)
	}
	deltaReverser := delFunc[param1].reverse
	dst = deltaReverser(dst)
	scaleReverser := scaleFunc[param3].reverse
	dst = scaleReverser(dst, minNum, maxNum)
	rangedReverser := rangedFunc[param0].reverse
	dst = rangedReverser(dst, base)

	return dst, nil
}
