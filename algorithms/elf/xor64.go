package elf

import "math"

// ElfXORCompressor encodes 64-bit patterns per Java implementation.
type ElfXORCompressor struct {
	storedLeadingZeros  int
	storedTrailingZeros int
	storedVal           uint64
	first               bool
	sizeBits            int
	out                 *BitWriter
}

var leadingRepresentation64 = [...]uint16{
	0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 2, 2, 2, 2,
	3, 3, 4, 4, 5, 5, 6, 6,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
}

var leadingRound64 = [...]uint16{
	0, 0, 0, 0, 0, 0, 0, 0,
	8, 8, 8, 8, 12, 12, 12, 12,
	16, 16, 18, 18, 20, 20, 22, 22,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
}

const endSign64 = 0x7ff8000000000000 // Double.NaN bits

func NewElfXORCompressor() *ElfXORCompressor {
	return &ElfXORCompressor{
		storedLeadingZeros:  math.MaxInt32,
		storedTrailingZeros: math.MaxInt32,
		first:               true,
		out:                 NewBitWriter(10000),
	}
}

func (c *ElfXORCompressor) Output() *BitWriter { return c.out }

func (c *ElfXORCompressor) AddLong(value uint64) int {
	if c.first {
		return c.writeFirst(value)
	}
	return c.compressValue(value)
}

func (c *ElfXORCompressor) AddDouble(value float64) int {
	bits := math.Float64bits(value)
	return c.AddLong(bits)
}

func (c *ElfXORCompressor) writeFirst(value uint64) int {
	c.first = false
	c.storedVal = value
	trailingZeros := tz64(value)
	c.out.WriteInt(uint64(trailingZeros), 7)
	if trailingZeros < 64 {
		c.out.WriteLong(value>>uint(trailingZeros+1), 63-trailingZeros)
		c.sizeBits += 70 - trailingZeros
		return 70 - trailingZeros
	}
	c.sizeBits += 7
	return 7
}

func (c *ElfXORCompressor) Close() {
	c.AddLong(endSign64)
	c.out.WriteBit(false)
	c.out.Flush()
}

func (c *ElfXORCompressor) compressValue(value uint64) int {
	thisSize := 0
	xor := c.storedVal ^ value
	if xor == 0 {
		c.out.WriteInt(1, 2) // case 01
		c.sizeBits += 2
		thisSize += 2
	} else {
		leadingZeros := int(leadingRound64[lz64(xor)])
		trailingZeros := tz64(xor)
		if leadingZeros == c.storedLeadingZeros && trailingZeros >= c.storedTrailingZeros {
			centerBits := 64 - c.storedLeadingZeros - c.storedTrailingZeros
			len := 2 + centerBits
			if len > 64 {
				c.out.WriteInt(0, 2)
				c.out.WriteLong(xor>>uint(c.storedTrailingZeros), centerBits)
			} else {
				c.out.WriteLong(xor>>uint(c.storedTrailingZeros), len)
			}
			c.sizeBits += len
			thisSize += len
		} else {
			c.storedLeadingZeros = leadingZeros
			c.storedTrailingZeros = trailingZeros
			centerBits := 64 - c.storedLeadingZeros - c.storedTrailingZeros
			if centerBits <= 16 {
				// case 10: 2 flag bits already included in 9 written bits
				code := (((0x2 << 3) | int(leadingRepresentation64[c.storedLeadingZeros])) << 4) | (centerBits & 0xF)
				c.out.WriteInt(uint64(code), 9)
				c.out.WriteLong(xor>>uint(c.storedTrailingZeros+1), centerBits-1)
				c.sizeBits += 8 + centerBits
				thisSize += 8 + centerBits
			} else {
				// case 11
				code := (((0x3 << 3) | int(leadingRepresentation64[c.storedLeadingZeros])) << 6) | (centerBits & 0x3F)
				c.out.WriteInt(uint64(code), 11)
				c.out.WriteLong(xor>>uint(c.storedTrailingZeros+1), centerBits-1)
				c.sizeBits += 10 + centerBits
				thisSize += 10 + centerBits
			}
		}
		c.storedVal = value
	}
	return thisSize
}

func lz64(x uint64) int { return bitsLenLeadingZeros64(x) }
func tz64(x uint64) int { return bitsLenTrailingZeros64(x) }

// Fallback helpers without importing math/bits to keep file standalone
func bitsLenLeadingZeros64(x uint64) int {
	if x == 0 {
		return 64
	}
	n := 0
	for i := 63; i >= 0; i-- {
		if (x>>uint(i))&1 == 0 {
			n++
		} else {
			break
		}
	}
	return n
}

func bitsLenTrailingZeros64(x uint64) int {
	if x == 0 {
		return 64
	}
	n := 0
	for i := 0; i < 64; i++ {
		if (x>>uint(i))&1 == 0 {
			n++
		} else {
			break
		}
	}
	return n
}

// ---------------- Decompressor ----------------

type ElfXORDecompressor struct {
	storedVal           uint64
	storedLeadingZeros  int
	storedTrailingZeros int
	first               bool
	endOfStream         bool
	in                  *BitReader
}

var leadingRepresentationDecode64 = [...]int{0, 8, 12, 16, 18, 20, 22, 24}

func NewElfXORDecompressor(bs []byte) *ElfXORDecompressor {
	return &ElfXORDecompressor{
		first: true,
		in:    NewBitReader(bs),
	}
}

func (d *ElfXORDecompressor) Input() *BitReader { return d.in }

func (d *ElfXORDecompressor) ReadValue() (*float64, error) {
	if err := d.next(); err != nil {
		return nil, err
	}
	if d.endOfStream {
		return nil, nil
	}
	v := math.Float64frombits(d.storedVal)
	return &v, nil
}

func (d *ElfXORDecompressor) next() error {
	if d.first {
		d.first = false
		trailingZerosU, err := d.in.ReadInt(7)
		if err != nil {
			return err
		}
		trailingZeros := int(trailingZerosU)
		if trailingZeros < 64 {
			val, err := d.in.ReadLong(63 - trailingZeros)
			if err != nil {
				return err
			}
			d.storedVal = ((val << 1) + 1) << uint(trailingZeros)
		} else {
			d.storedVal = 0
		}
		if d.storedVal == endSign64 {
			d.endOfStream = true
		}
		return nil
	}
	return d.nextValue()
}

func (d *ElfXORDecompressor) nextValue() error {
	flagU, err := d.in.ReadInt(2)
	if err != nil {
		return err
	}
	flag := int(flagU)
	switch flag {
	case 3: // 11
		leadAndCenterU, err := d.in.ReadInt(9)
		if err != nil {
			return err
		}
		leadAndCenter := int(leadAndCenterU)
		d.storedLeadingZeros = leadingRepresentationDecode64[leadAndCenter>>6]
		centerBits := leadAndCenter & 0x3F
		if centerBits == 0 {
			centerBits = 64
		}
		d.storedTrailingZeros = 64 - d.storedLeadingZeros - centerBits
		val, err := d.in.ReadLong(centerBits - 1)
		if err != nil {
			return err
		}
		value := ((val << 1) + 1) << uint(d.storedTrailingZeros)
		value ^= d.storedVal
		if value == endSign64 {
			d.endOfStream = true
		} else {
			d.storedVal = value
		}
	case 2: // 10
		leadAndCenterU, err := d.in.ReadInt(7)
		if err != nil {
			return err
		}
		leadAndCenter := int(leadAndCenterU)
		d.storedLeadingZeros = leadingRepresentationDecode64[leadAndCenter>>4]
		centerBits := leadAndCenter & 0xF
		if centerBits == 0 {
			centerBits = 16
		}
		d.storedTrailingZeros = 64 - d.storedLeadingZeros - centerBits
		val, err := d.in.ReadLong(centerBits - 1)
		if err != nil {
			return err
		}
		value := ((val << 1) + 1) << uint(d.storedTrailingZeros)
		value ^= d.storedVal
		if value == endSign64 {
			d.endOfStream = true
		} else {
			d.storedVal = value
		}
	case 1: // 01 (no-op)
		// same value
	default: // 00
		centerBits := 64 - d.storedLeadingZeros - d.storedTrailingZeros
		val, err := d.in.ReadLong(centerBits)
		if err != nil {
			return err
		}
		value := val << uint(d.storedTrailingZeros)
		value ^= d.storedVal
		if value == endSign64 {
			d.endOfStream = true
		} else {
			d.storedVal = value
		}
	}
	return nil
}
