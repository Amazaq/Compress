package chimp128

import (
	"fmt"
	"math"
	"math/bits"
)

// OutputBitStream handles bit-level writing operations
type OutputBitStream struct {
	buffer   []byte
	bitPos   int
	bytePos  int
	capacity int
}

// NewOutputBitStream creates a new output bit stream with given capacity
func NewOutputBitStream(capacity int) *OutputBitStream {
	return &OutputBitStream{
		buffer:   make([]byte, capacity),
		bitPos:   0,
		bytePos:  0,
		capacity: capacity,
	}
}

// WriteBit writes a single bit
func (obs *OutputBitStream) WriteBit(bit bool) {
	if obs.bytePos >= obs.capacity {
		// Expand buffer if needed
		newCapacity := obs.capacity * 2
		newBuffer := make([]byte, newCapacity)
		copy(newBuffer, obs.buffer)
		obs.buffer = newBuffer
		obs.capacity = newCapacity
	}

	if bit {
		obs.buffer[obs.bytePos] |= 1 << (7 - obs.bitPos)
	}

	obs.bitPos++
	if obs.bitPos == 8 {
		obs.bitPos = 0
		obs.bytePos++
	}
}

// WriteInt writes an integer with specified number of bits
func (obs *OutputBitStream) WriteInt(value int, bits int) {
	for i := bits - 1; i >= 0; i-- {
		obs.WriteBit((value>>i)&1 == 1)
	}
}

// WriteLong writes a long integer with specified number of bits
func (obs *OutputBitStream) WriteLong(value int64, bits int) {
	for i := bits - 1; i >= 0; i-- {
		obs.WriteBit((value>>i)&1 == 1)
	}
}

// Flush ensures all bits are written
func (obs *OutputBitStream) Flush() {
	if obs.bitPos > 0 {
		obs.bytePos++
	}
}

// GetBuffer returns the written bytes
func (obs *OutputBitStream) GetBuffer() []byte {
	return obs.buffer[:obs.bytePos]
}

// InputBitStream handles bit-level reading operations
type InputBitStream struct {
	buffer  []byte
	bitPos  int
	bytePos int
	length  int
}

// NewInputBitStream creates a new input bit stream
func NewInputBitStream(data []byte) *InputBitStream {
	return &InputBitStream{
		buffer:  data,
		bitPos:  0,
		bytePos: 0,
		length:  len(data),
	}
}

// ReadBit reads a single bit
func (ibs *InputBitStream) ReadBit() (bool, error) {
	if ibs.bytePos >= ibs.length {
		return false, fmt.Errorf("EOF")
	}

	bit := (ibs.buffer[ibs.bytePos] >> (7 - ibs.bitPos)) & 1
	ibs.bitPos++

	if ibs.bitPos == 8 {
		ibs.bitPos = 0
		ibs.bytePos++
	}

	return bit == 1, nil
}

// ReadInt reads an integer with specified number of bits
func (ibs *InputBitStream) ReadInt(bits int) (int, error) {
	value := 0
	for i := 0; i < bits; i++ {
		bit, err := ibs.ReadBit()
		if err != nil {
			return 0, err
		}
		if bit {
			value |= 1 << (bits - 1 - i)
		}
	}
	return value, nil
}

// ReadLong reads a long integer with specified number of bits
func (ibs *InputBitStream) ReadLong(bits int) (int64, error) {
	var value int64 = 0
	for i := 0; i < bits; i++ {
		bit, err := ibs.ReadBit()
		if err != nil {
			return 0, err
		}
		if bit {
			value |= int64(1) << (bits - 1 - i)
		}
	}
	return value, nil
}

// ChimpN implements the Chimp128 time series compression for floating points
type ChimpN struct {
	storedLeadingZeros int
	storedValues       []int64
	first              bool
	size               int
	previousValuesLog2 int
	threshold          int
	out                *OutputBitStream
	previousValues     int
	setLsb             int
	indices            []int
	index              int
	current            int
	flagOneSize        int
	flagZeroSize       int
}

// Static lookup tables
var leadingRepresentation = []int16{
	0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 2, 2, 2, 2,
	3, 3, 4, 4, 5, 5, 6, 6,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
}

var leadingRound = []int16{
	0, 0, 0, 0, 0, 0, 0, 0,
	8, 8, 8, 8, 12, 12, 12, 12,
	16, 16, 18, 18, 20, 20, 22, 22,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
}

// NewChimpN creates a new ChimpN compressor
func NewChimpN(previousValues int) *ChimpN {
	previousValuesLog2 := int(math.Log2(float64(previousValues)))
	threshold := 6 + previousValuesLog2
	setLsb := int(math.Pow(2, float64(threshold+1))) - 1

	return &ChimpN{
		storedLeadingZeros: math.MaxInt32,
		first:              true,
		previousValues:     previousValues,
		previousValuesLog2: previousValuesLog2,
		threshold:          threshold,
		setLsb:             setLsb,
		indices:            make([]int, int(math.Pow(2, float64(threshold+1)))),
		storedValues:       make([]int64, previousValues),
		flagZeroSize:       previousValuesLog2 + 2,
		flagOneSize:        previousValuesLog2 + 11,
		out:                NewOutputBitStream(1000 * 8),
		index:              0,
		current:            0,
		size:               0,
	}
}

// GetOut returns the compressed data buffer
func (cn *ChimpN) GetOut() []byte {
	return cn.out.GetBuffer()
}

// AddValueLong adds a new long value to the series
func (cn *ChimpN) AddValueLong(value int64) {
	if cn.first {
		cn.writeFirst(value)
	} else {
		cn.compressValue(value)
	}
}

// AddValueDouble adds a new double value to the series
func (cn *ChimpN) AddValueDouble(value float64) {
	longBits := int64(math.Float64bits(value))
	if cn.first {
		cn.writeFirst(longBits)
	} else {
		cn.compressValue(longBits)
	}
}

func (cn *ChimpN) writeFirst(value int64) {
	cn.first = false
	cn.storedValues[cn.current] = value
	cn.out.WriteLong(cn.storedValues[cn.current], 64)
	cn.indices[int(value)&cn.setLsb] = cn.index
	cn.size += 64
}

// Close closes the block and writes the remaining data
func (cn *ChimpN) Close() {
	cn.AddValueDouble(math.NaN())
	cn.out.Flush()
}

func (cn *ChimpN) compressValue(value int64) {
	key := int(value) & cn.setLsb
	var xor int64
	var previousIndex int
	trailingZeros := 0
	currIndex := cn.indices[key]

	if (cn.index - currIndex) < cn.previousValues {
		tempXor := value ^ cn.storedValues[currIndex%cn.previousValues]
		trailingZeros = bits.TrailingZeros64(uint64(tempXor))
		if trailingZeros > cn.threshold {
			previousIndex = currIndex % cn.previousValues
			xor = tempXor
		} else {
			previousIndex = cn.index % cn.previousValues
			xor = cn.storedValues[previousIndex] ^ value
		}
	} else {
		previousIndex = cn.index % cn.previousValues
		xor = cn.storedValues[previousIndex] ^ value
	}

	if xor == 0 {
		// Same value case - write flag=0 directly in the combined format
		cn.out.WriteInt(previousIndex, cn.flagZeroSize)
		cn.size += cn.flagZeroSize
	} else {
		leadingZeros := int(leadingRound[bits.LeadingZeros64(uint64(xor))])

		if trailingZeros > cn.threshold {
			// Trailing zeros case - this maps to Java's case 1
			significantBits := 64 - leadingZeros - trailingZeros

			// Build the flag value exactly as Java expects for case 1
			flagValue := 512*(cn.previousValues+previousIndex) + 64*int(leadingRepresentation[leadingZeros]) + significantBits
			cn.out.WriteInt(flagValue, cn.flagOneSize)
			cn.out.WriteLong(xor>>trailingZeros, significantBits)
			cn.size += significantBits + cn.flagOneSize
			cn.storedLeadingZeros = 65
		} else if leadingZeros == cn.storedLeadingZeros {
			// Same leading zeros case - maps to Java's case 2
			cn.out.WriteInt(2, 2)
			significantBits := 64 - leadingZeros
			cn.out.WriteLong(xor, significantBits)
			cn.size += 2 + significantBits
		} else {
			// New leading zeros case - maps to Java's case 3
			cn.storedLeadingZeros = leadingZeros
			significantBits := 64 - leadingZeros
			cn.out.WriteInt(24+int(leadingRepresentation[leadingZeros]), 5)
			cn.out.WriteLong(xor, significantBits)
			cn.size += 5 + significantBits
		}
	}

	cn.current = (cn.current + 1) % cn.previousValues
	cn.storedValues[cn.current] = value
	cn.index++
	cn.indices[key] = cn.index
}

// GetSize returns the total size in bits
func (cn *ChimpN) GetSize() int {
	return cn.size
}

// ChimpNDecompressor implements the decompression logic - based on Java implementation
type ChimpNDecompressor struct {
	storedLeadingZeros  int
	storedTrailingZeros int
	storedVal           int64
	storedValues        []int64
	current             int
	first               bool
	endOfStream         bool
	in                  *InputBitStream
	previousValues      int
	previousValuesLog2  int
	initialFill         int
}

// Static lookup table for decompressor (matches Java implementation)
var decompLeadingRepresentation = []int16{0, 8, 12, 16, 18, 20, 22, 24}

// NewChimpNDecompressor creates a new decompressor
func NewChimpNDecompressor(data []byte, previousValues int) *ChimpNDecompressor {
	previousValuesLog2 := int(math.Log2(float64(previousValues)))

	return &ChimpNDecompressor{
		storedLeadingZeros:  math.MaxInt32,
		storedTrailingZeros: 0,
		storedVal:           0,
		current:             0,
		first:               true,
		endOfStream:         false,
		in:                  NewInputBitStream(data),
		previousValues:      previousValues,
		previousValuesLog2:  previousValuesLog2,
		initialFill:         previousValuesLog2 + 9,
		storedValues:        make([]int64, previousValues),
	}
}

// ReadValue reads the next decompressed double value
func (cd *ChimpNDecompressor) ReadValue() (*float64, error) {
	err := cd.next()
	if err != nil {
		return nil, err
	}
	if cd.endOfStream {
		return nil, nil
	}
	result := math.Float64frombits(uint64(cd.storedVal))
	return &result, nil
}

func (cd *ChimpNDecompressor) next() error {
	if cd.first {
		cd.first = false
		val, err := cd.in.ReadLong(64)
		if err != nil {
			return err
		}
		cd.storedVal = val
		cd.storedValues[cd.current] = cd.storedVal
		if math.IsNaN(math.Float64frombits(uint64(cd.storedVal))) {
			cd.endOfStream = true
		}
	} else {
		return cd.nextValue()
	}
	return nil
}

func (cd *ChimpNDecompressor) nextValue() error {
	// Read the flag exactly as Java implementation expects
	flag, err := cd.in.ReadInt(2)
	if err != nil {
		return err
	}

	var value int64

	switch flag {
	case 3: // New leading zeros
		leadingIndex, err := cd.in.ReadInt(3)
		if err != nil {
			return err
		}
		cd.storedLeadingZeros = int(decompLeadingRepresentation[leadingIndex])
		val, err := cd.in.ReadLong(64 - cd.storedLeadingZeros)
		if err != nil {
			return err
		}
		value = cd.storedVal ^ val

		if math.IsNaN(math.Float64frombits(uint64(value))) {
			cd.endOfStream = true
			return nil
		} else {
			cd.storedVal = value
			cd.current = (cd.current + 1) % cd.previousValues
			cd.storedValues[cd.current] = cd.storedVal
		}

	case 2: // Same leading zeros
		val, err := cd.in.ReadLong(64 - cd.storedLeadingZeros)
		if err != nil {
			return err
		}
		value = cd.storedVal ^ val

		if math.IsNaN(math.Float64frombits(uint64(value))) {
			cd.endOfStream = true
			return nil
		} else {
			cd.storedVal = value
			cd.current = (cd.current + 1) % cd.previousValues
			cd.storedValues[cd.current] = cd.storedVal
		}

	case 1: // Trailing zeros case
		fill := cd.initialFill
		temp, err := cd.in.ReadInt(fill)
		if err != nil {
			return err
		}

		// Extract index (Java: temp >>> (fill -= previousValuesLog2) & (1 << previousValuesLog2) - 1)
		fill -= cd.previousValuesLog2
		index := (temp >> fill) & ((1 << cd.previousValuesLog2) - 1)

		// Extract leading zeros (Java: temp >>> (fill -= 3) & (1 << 3) - 1)
		fill -= 3
		leadingIndex := (temp >> fill) & ((1 << 3) - 1)
		cd.storedLeadingZeros = int(decompLeadingRepresentation[leadingIndex])

		// Extract significant bits (Java: temp >>> (fill -= 6) & (1 << 6) - 1)
		fill -= 6
		significantBits := (temp >> fill) & ((1 << 6) - 1)

		cd.storedVal = cd.storedValues[index]
		if significantBits == 0 {
			significantBits = 64
		}

		cd.storedTrailingZeros = 64 - significantBits - cd.storedLeadingZeros
		val, err := cd.in.ReadLong(64 - cd.storedLeadingZeros - cd.storedTrailingZeros)
		if err != nil {
			return err
		}

		val <<= cd.storedTrailingZeros
		value = cd.storedVal ^ val

		if math.IsNaN(math.Float64frombits(uint64(value))) {
			cd.endOfStream = true
			return nil
		} else {
			cd.storedVal = value
			cd.current = (cd.current + 1) % cd.previousValues
			cd.storedValues[cd.current] = cd.storedVal
		}

	default: // case 0: Same value as before
		// Read remaining bits to get the index
		index, err := cd.in.ReadLong(cd.previousValuesLog2)
		if err != nil {
			return err
		}
		cd.storedVal = cd.storedValues[index]
		cd.current = (cd.current + 1) % cd.previousValues
		cd.storedValues[cd.current] = cd.storedVal
	}

	return nil
}

// CompressFloat compresses a float64 array using Chimp128 algorithm
func CompressFloat(dst []byte, src []float64) []byte {
	if len(src) == 0 {
		return dst
	}

	// Use 128 previous values for Chimp128
	chimp := NewChimpN(128)

	// Compress all values
	for _, v := range src {
		chimp.AddValueDouble(v)
	}

	chimp.Close()
	compressed := chimp.GetOut()

	// Append to dst if provided, otherwise return new slice
	if dst == nil {
		return compressed
	}

	return append(dst, compressed...)
}

// DecompressFloat decompresses a byte array back to float64 array using Chimp128 algorithm
func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	if len(src) == 0 {
		if dst == nil {
			return []float64{}, nil
		}
		return dst[:0], nil
	}

	decompressor := NewChimpNDecompressor(src, 128)
	var result []float64

	// Use dst as base if provided
	if dst != nil {
		result = dst[:0]
	}

	for {
		value, err := decompressor.ReadValue()
		if err != nil {
			return result, fmt.Errorf("decompression error: %v", err)
		}

		// Check for end of stream (nil return)
		if value == nil {
			break
		}

		result = append(result, *value)
	}

	return result, nil
}

// Example usage and test
func main() {
	// Test data
	original := []float64{1.23, 1.24, 1.25, 1.26, 1.27, 3.14159, 2.71828, 1.41421}

	fmt.Printf("Original values: %v\n", original)

	// Compress
	compressed := CompressFloat(nil, original)
	fmt.Printf("Original size: %d bytes\n", len(original)*8)
	fmt.Printf("Compressed size: %d bytes\n", len(compressed))
	fmt.Printf("Compression ratio: %.2f\n", float64(len(original)*8)/float64(len(compressed)))

	// Decompress
	decompressed, err := DecompressFloat(nil, compressed)
	if err != nil {
		fmt.Printf("Decompression error: %v\n", err)
		return
	}

	// Verify
	fmt.Printf("Decompressed %d values: %v\n", len(decompressed), decompressed)

	// Check if values match
	allMatch := len(original) == len(decompressed)
	for i := 0; i < len(original) && i < len(decompressed); i++ {
		if original[i] != decompressed[i] {
			allMatch = false
			break
		}
	}
	fmt.Printf("All values match: %v\n", allMatch)
}
