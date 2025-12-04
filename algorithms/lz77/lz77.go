package lz77

import (
	"encoding/binary"
	"errors"
	"math"
)

const (
	maxWindowSize    = 1 << 15
	minMatchLength   = 3
	maxMatchLength   = math.MaxUint16
	maxLiteralLength = math.MaxUint16
)

const (
	tokenLiteral byte = 0
	tokenMatch   byte = 1
)

const (
	symbolTokenLiteral uint64 = 0
	symbolTokenMatch   uint64 = 1
)

func CompressFloat(dst []byte, src []float64) []byte {
	if len(src) == 0 {
		return dst[:0]
	}
	raw := make([]byte, len(src)*8)
	for i, v := range src {
		binary.LittleEndian.PutUint64(raw[i*8:(i+1)*8], math.Float64bits(v))
	}
	encoded := compressBytes(raw)
	return append(dst[:0], encoded...)
}

func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}
	decoded, err := decompressBytes(src)
	if err != nil {
		return nil, err
	}
	if len(decoded)%8 != 0 {
		return nil, errors.New("lz77: decoded byte stream not aligned to float64")
	}
	count := len(decoded) / 8
	dst = append(dst[:0], make([]float64, count)...)
	for i := 0; i < count; i++ {
		bits := binary.LittleEndian.Uint64(decoded[i*8 : (i+1)*8])
		dst[i] = math.Float64frombits(bits)
	}
	return dst, nil
}

func Compress(dst []byte, src []uint64) []byte {
	if len(src) == 0 {
		return dst[:0]
	}
	raw := make([]byte, len(src)*8)
	for i, v := range src {
		binary.LittleEndian.PutUint64(raw[i*8:(i+1)*8], v)
	}
	encoded := compressBytes(raw)
	return append(dst[:0], encoded...)
}

func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}
	decoded, err := decompressBytes(src)
	if err != nil {
		return nil, err
	}
	if len(decoded)%8 != 0 {
		return nil, errors.New("lz77: decoded byte stream not aligned to uint64")
	}
	count := len(decoded) / 8
	dst = append(dst[:0], make([]uint64, count)...)
	for i := 0; i < count; i++ {
		dst[i] = binary.LittleEndian.Uint64(decoded[i*8 : (i+1)*8])
	}
	return dst, nil
}

func compressBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	encoded := make([]byte, 0, len(data))
	pos := 0
	for pos < len(data) {
		offset, length := findLongestMatch(data, pos)
		if length >= minMatchLength {
			encoded = append(encoded, tokenMatch)
			encoded = appendUint16(encoded, uint16(offset))
			encoded = appendUint16(encoded, uint16(length))
			pos += length
			continue
		}

		literalStart := pos
		pos++
		for pos < len(data) {
			if pos-literalStart >= maxLiteralLength {
				break
			}
			_, nextLength := findLongestMatch(data, pos)
			if nextLength >= minMatchLength {
				break
			}
			pos++
		}
		literalLen := pos - literalStart
		encoded = append(encoded, tokenLiteral)
		encoded = appendUint16(encoded, uint16(literalLen))
		encoded = append(encoded, data[literalStart:pos]...)
	}
	return encoded
}

func decompressBytes(encoded []byte) ([]byte, error) {
	out := make([]byte, 0, len(encoded)*2)
	cursor := 0
	for cursor < len(encoded) {
		token := encoded[cursor]
		cursor++
		switch token {
		case tokenLiteral:
			if cursor+2 > len(encoded) {
				return nil, errors.New("lz77: truncated literal length")
			}
			literalLen := int(binary.LittleEndian.Uint16(encoded[cursor : cursor+2]))
			cursor += 2
			if cursor+literalLen > len(encoded) {
				return nil, errors.New("lz77: truncated literal data")
			}
			out = append(out, encoded[cursor:cursor+literalLen]...)
			cursor += literalLen
		case tokenMatch:
			if cursor+4 > len(encoded) {
				return nil, errors.New("lz77: truncated match header")
			}
			offset := int(binary.LittleEndian.Uint16(encoded[cursor : cursor+2]))
			cursor += 2
			length := int(binary.LittleEndian.Uint16(encoded[cursor : cursor+2]))
			cursor += 2
			if offset <= 0 || offset > len(out) {
				return nil, errors.New("lz77: invalid match offset")
			}
			if length <= 0 {
				return nil, errors.New("lz77: invalid match length")
			}
			start := len(out) - offset
			for i := 0; i < length; i++ {
				out = append(out, out[start+i])
			}
		default:
			return nil, errors.New("lz77: unknown token type")
		}
	}
	return out, nil
}

// CompressSymbols performs LZ77 compression treating each uint64 as a single symbol
// so the original array structure is preserved.
func CompressSymbols(dst []uint64, src []uint64) []uint64 {
	if len(src) == 0 {
		return dst[:0]
	}
	encoded := make([]uint64, 0, len(src))
	pos := 0
	for pos < len(src) {
		offset, length := findLongestMatchUint64(src, pos)
		if length >= minMatchLength {
			encoded = append(encoded, symbolTokenMatch, uint64(offset), uint64(length))
			pos += length
			continue
		}

		literalStart := pos
		pos++
		for pos < len(src) {
			if pos-literalStart >= maxLiteralLength {
				break
			}
			_, nextLength := findLongestMatchUint64(src, pos)
			if nextLength >= minMatchLength {
				break
			}
			pos++
		}
		literalLen := pos - literalStart
		encoded = append(encoded, symbolTokenLiteral, uint64(literalLen))
		encoded = append(encoded, src[literalStart:pos]...)
	}
	return append(dst[:0], encoded...)
}

// DecompressSymbols reverses CompressSymbols, producing the original uint64 slice.
func DecompressSymbols(dst []uint64, encoded []uint64) ([]uint64, error) {
	out := make([]uint64, 0, len(encoded))
	cursor := 0
	for cursor < len(encoded) {
		token := encoded[cursor]
		cursor++
		switch token {
		case symbolTokenLiteral:
			if cursor >= len(encoded) {
				return nil, errors.New("lz77: truncated literal length (symbols)")
			}
			literalLen := int(encoded[cursor])
			cursor++
			if literalLen < 0 || cursor+literalLen > len(encoded) {
				return nil, errors.New("lz77: truncated literal data (symbols)")
			}
			out = append(out, encoded[cursor:cursor+literalLen]...)
			cursor += literalLen
		case symbolTokenMatch:
			if cursor+1 >= len(encoded) {
				return nil, errors.New("lz77: truncated match header (symbols)")
			}
			offset := int(encoded[cursor])
			length := int(encoded[cursor+1])
			cursor += 2
			if offset <= 0 || offset > len(out) {
				return nil, errors.New("lz77: invalid match offset (symbols)")
			}
			if length <= 0 {
				return nil, errors.New("lz77: invalid match length (symbols)")
			}
			start := len(out) - offset
			for i := 0; i < length; i++ {
				out = append(out, out[start+i])
			}
		default:
			return nil, errors.New("lz77: unknown token type (symbols)")
		}
	}
	return append(dst[:0], out...), nil
}

func findLongestMatchUint64(data []uint64, pos int) (offset, length int) {
	if pos == 0 {
		return 0, 0
	}
	windowStart := pos - maxWindowSize
	if windowStart < 0 {
		windowStart = 0
	}
	bestOffset := 0
	bestLength := 0
	limit := len(data)
	for i := windowStart; i < pos; i++ {
		currentOffset := pos - i
		if currentOffset > math.MaxUint16 {
			continue
		}
		matchLen := 0
		for matchLen < maxMatchLength && pos+matchLen < limit && data[i+matchLen] == data[pos+matchLen] {
			matchLen++
		}
		if matchLen > bestLength {
			bestLength = matchLen
			bestOffset = currentOffset
			if bestLength == maxMatchLength {
				break
			}
		}
	}
	if bestLength < minMatchLength {
		return 0, 0
	}
	return bestOffset, bestLength
}

func appendUint16(dst []byte, value uint16) []byte {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], value)
	return append(dst, tmp[:]...)
}

func findLongestMatch(data []byte, pos int) (offset, length int) {
	if pos == 0 {
		return 0, 0
	}
	windowStart := pos - maxWindowSize
	if windowStart < 0 {
		windowStart = 0
	}
	bestOffset := 0
	bestLength := 0
	limit := len(data)
	for i := windowStart; i < pos; i++ {
		currentOffset := pos - i
		if currentOffset > math.MaxUint16 {
			continue
		}
		matchLen := 0
		for matchLen < maxMatchLength && pos+matchLen < limit && data[i+matchLen] == data[pos+matchLen] {
			matchLen++
		}
		if matchLen > bestLength {
			bestLength = matchLen
			bestOffset = currentOffset
			if bestLength == maxMatchLength {
				break
			}
		}
	}
	if bestLength < minMatchLength {
		return 0, 0
	}
	return bestOffset, bestLength
}
