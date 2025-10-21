package huffman

import (
	"container/heap"
	"encoding/binary"
	"fmt"
	"math"
)

// HuffmanNode 表示Huffman树的节点
type HuffmanNode struct {
	value  float64
	freq   int
	left   *HuffmanNode
	right  *HuffmanNode
	isLeaf bool
}

// PriorityQueue 实现优先队列用于构建Huffman树
type PriorityQueue []*HuffmanNode

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].freq < pq[j].freq
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*HuffmanNode))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// buildFrequencyTable 构建浮点数频率表
func buildFrequencyTable(data []float64) map[float64]int {
	freq := make(map[float64]int)
	for _, f := range data {
		freq[f]++
	}
	return freq
}

// buildHuffmanTree 构建Huffman树
func buildHuffmanTree(freq map[float64]int) *HuffmanNode {
	if len(freq) == 0 {
		return nil
	}
	// 如果只有一个不同的浮点数，创建特殊情况
	if len(freq) == 1 {
		for f, count := range freq {
			return &HuffmanNode{
				value:  f,
				freq:   count,
				isLeaf: true,
			}
		}
	}

	pq := &PriorityQueue{}
	heap.Init(pq)

	// 为每个浮点数创建叶子节点
	for f, count := range freq {
		node := &HuffmanNode{
			value:  f,
			freq:   count,
			isLeaf: true,
		}
		heap.Push(pq, node)
	}

	// 构建Huffman树
	for pq.Len() > 1 {
		left := heap.Pop(pq).(*HuffmanNode)
		right := heap.Pop(pq).(*HuffmanNode)

		merged := &HuffmanNode{
			freq:   left.freq + right.freq,
			left:   left,
			right:  right,
			isLeaf: false,
		}
		heap.Push(pq, merged)
	}

	return heap.Pop(pq).(*HuffmanNode)
}

// generateCodes 生成Huffman编码表
func generateCodes(root *HuffmanNode) map[float64]string {
	if root == nil {
		return nil
	}

	codes := make(map[float64]string)

	// 特殊情况：只有一个不同的浮点数
	if root.isLeaf {
		codes[root.value] = "0"
		return codes
	}

	var generate func(*HuffmanNode, string)
	generate = func(node *HuffmanNode, code string) {
		if node.isLeaf {
			codes[node.value] = code
			return
		}
		if node.left != nil {
			generate(node.left, code+"0")
		}
		if node.right != nil {
			generate(node.right, code+"1")
		}
	}

	generate(root, "")
	return codes
}

// encodeBits 将比特字符串转换为字节数组
func encodeBits(bits string) []byte {
	// 计算需要的字节数
	byteCount := (len(bits) + 7) / 8
	result := make([]byte, byteCount)

	for i, bit := range bits {
		if bit == '1' {
			byteIndex := i / 8
			bitIndex := 7 - (i % 8)
			result[byteIndex] |= 1 << bitIndex
		}
	}

	return result
}

// serializeTree 序列化Huffman树以便解压时重建
func serializeTree(root *HuffmanNode) []byte {
	if root == nil {
		return []byte{}
	}

	var result []byte
	var serialize func(*HuffmanNode)
	serialize = func(node *HuffmanNode) {
		if node.isLeaf {
			result = append(result, 1) // 标记叶子节点
			// 将float64转换为字节存储
			bits := math.Float64bits(node.value)
			valueBytes := make([]byte, 8)
			binary.LittleEndian.PutUint64(valueBytes, bits)
			result = append(result, valueBytes...)
		} else {
			result = append(result, 0) // 标记内部节点
			serialize(node.left)
			serialize(node.right)
		}
	}

	serialize(root)
	return result
}

// CompressFloat 使用Huffman编码压缩浮点数数组
func CompressFloat(dst []byte, src []float64) []byte {
	if len(src) == 0 {
		return dst
	}
	// 1. 构建浮点数频率表
	freq := buildFrequencyTable(src)

	// 2. 构建Huffman树
	root := buildHuffmanTree(freq)
	if root == nil {
		return dst
	}
	// 3. 生成编码表
	codes := generateCodes(root)

	// 4. 编码数据
	var encodedBits string
	for _, f := range src {
		encodedBits += codes[f]
	}

	// 5. 序列化结果
	// 格式: [原始长度(4字节)] + [树大小(4字节)] + [序列化的树] + [编码长度(4字节)] + [编码数据]

	// 原始数据长度
	originalLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(originalLen, uint32(len(src)))

	// 序列化树
	serializedTree := serializeTree(root)
	treeSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(treeSize, uint32(len(serializedTree)))

	// 编码数据
	encodedData := encodeBits(encodedBits)
	encodedLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(encodedLen, uint32(len(encodedBits)))

	// 组装结果
	dst = append(dst, originalLen...)
	dst = append(dst, treeSize...)
	dst = append(dst, serializedTree...)
	dst = append(dst, encodedLen...)
	dst = append(dst, encodedData...)

	return dst
}

// deserializeTree 从字节数据重建Huffman树
func deserializeTree(data []byte) (*HuffmanNode, int, error) {
	if len(data) == 0 {
		return nil, 0, fmt.Errorf("empty tree data")
	}

	offset := 0
	var deserialize func() (*HuffmanNode, error)
	deserialize = func() (*HuffmanNode, error) {
		if offset >= len(data) {
			return nil, fmt.Errorf("unexpected end of tree data")
		}

		nodeType := data[offset]
		offset++

		if nodeType == 1 { // 叶子节点
			if offset+8 > len(data) {
				return nil, fmt.Errorf("insufficient data for leaf node")
			}

			bits := binary.LittleEndian.Uint64(data[offset : offset+8])
			value := math.Float64frombits(bits)
			offset += 8

			return &HuffmanNode{
				value:  value,
				isLeaf: true,
			}, nil
		} else { // 内部节点
			left, err := deserialize()
			if err != nil {
				return nil, err
			}

			right, err := deserialize()
			if err != nil {
				return nil, err
			}

			return &HuffmanNode{
				left:   left,
				right:  right,
				isLeaf: false,
			}, nil
		}
	}

	root, err := deserialize()
	return root, offset, err
}

// DecompressFloat 解压缩浮点数数组
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	if len(src) == 0 {
		return dst, nil
	}

	offset := 0

	// 1. 读取原始长度
	if offset+4 > len(src) {
		return nil, fmt.Errorf("insufficient data for original length")
	}
	originalLen := binary.LittleEndian.Uint32(src[offset : offset+4])
	offset += 4

	// 2. 读取树大小
	if offset+4 > len(src) {
		return nil, fmt.Errorf("insufficient data for tree size")
	}
	treeSize := binary.LittleEndian.Uint32(src[offset : offset+4])
	offset += 4

	// 3. 反序列化Huffman树
	if offset+int(treeSize) > len(src) {
		return nil, fmt.Errorf("insufficient data for tree")
	}
	root, _, err := deserializeTree(src[offset : offset+int(treeSize)])
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize tree: %v", err)
	}
	offset += int(treeSize)

	// 4. 读取编码长度
	if offset+4 > len(src) {
		return nil, fmt.Errorf("insufficient data for encoded length")
	}
	encodedLen := binary.LittleEndian.Uint32(src[offset : offset+4])
	offset += 4

	// 5. 读取编码数据
	encodedDataSize := (encodedLen + 7) / 8 // 计算字节数
	if offset+int(encodedDataSize) > len(src) {
		return nil, fmt.Errorf("insufficient data for encoded data")
	}
	encodedData := src[offset : offset+int(encodedDataSize)]

	// 6. 解码数据
	result := dst
	if cap(result)-len(result) < int(originalLen) {
		newResult := make([]float64, len(result), len(result)+int(originalLen))
		copy(newResult, result)
		result = newResult
	}

	// 特殊情况：只有一个不同的浮点数
	if root.isLeaf {
		for i := 0; i < int(originalLen); i++ {
			result = append(result, root.value)
		}
		return result, nil
	}

	// 解码比特流
	current := root
	bitIndex := 0

	for len(result)-len(dst) < int(originalLen) && bitIndex < int(encodedLen) {
		// 读取当前比特
		byteIndex := bitIndex / 8
		bitPos := 7 - (bitIndex % 8)
		bit := (encodedData[byteIndex] >> bitPos) & 1
		bitIndex++

		// 根据比特移动到左子树或右子树
		if bit == 0 {
			current = current.left
		} else {
			current = current.right
		}

		// 如果到达叶子节点，输出值并重置到根节点
		if current.isLeaf {
			result = append(result, current.value)
			current = root
		}
	}

	// 检查是否解码了预期数量的浮点数
	if len(result)-len(dst) != int(originalLen) {
		return nil, fmt.Errorf("decoded %d values, expected %d",
			len(result)-len(dst), originalLen)
	}

	return result, nil
}
