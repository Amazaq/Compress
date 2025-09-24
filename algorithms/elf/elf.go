package elf

import "math"

// -------- Top-level compressor (erasure + XOR) --------

type Compressor struct {
	xor          *ElfXORCompressor
	sizeBits     int
	lastBetaStar int
}

func NewCompressor() *Compressor {
	return &Compressor{xor: NewElfXORCompressor(), lastBetaStar: math.MaxInt32}
}

// Add adds a double value following Java AbstractElfCompressor + ElfCompressor logic.
func (c *Compressor) Add(v float64) {
	vLong := math.Float64bits(v)
	var vPrime uint64
	if v == 0.0 || math.IsInf(v, 0) {
		c.sizeBits += c.writeInt(2, 2)
		vPrime = vLong
	} else if math.IsNaN(v) {
		c.sizeBits += c.writeInt(2, 2)
		vPrime = math.Float64bits(math.NaN()) // 0x7ff8...
	} else {
		alpha, betaStar := GetAlphaAndBetaStar(v, c.lastBetaStar)
		e := int((vLong >> 52) & 0x7ff)
		gAlpha := GetFAlpha(alpha) + e - 1023
		eraseBits := 52 - gAlpha
		mask := ^uint64(0) << uint(eraseBits)
		delta := (^mask) & vLong
		if delta != 0 && eraseBits > 4 {
			if betaStar == c.lastBetaStar {
				c.sizeBits += c.writeBit(false) // case 0
			} else {
				c.sizeBits += c.writeInt(uint64(betaStar)|0x30, 6) // case 11 (2+4)
				c.lastBetaStar = betaStar
			}
			vPrime = mask & vLong
		} else {
			c.sizeBits += c.writeInt(2, 2) // case 10
			vPrime = vLong
		}
	}
	c.sizeBits += c.xor.AddLong(vPrime)
}

func (c *Compressor) writeInt(n uint64, len int) int { c.xor.out.WriteInt(n, len); return len }
func (c *Compressor) writeBit(bit bool) int          { c.xor.out.WriteBit(bit); return 1 }

func (c *Compressor) Bytes() []byte { return c.xor.out.Bytes() }

func (c *Compressor) Close() {
	// Write one more case 10 per Java close
	c.writeInt(2, 2)
	c.xor.Close()
}

// -------- Top-level decompressor --------

type Decompressor struct {
	xor          *ElfXORDecompressor
	lastBetaStar int
}

func NewDecompressor(bs []byte) *Decompressor {
	return &Decompressor{xor: NewElfXORDecompressor(bs), lastBetaStar: math.MaxInt32}
}

func (d *Decompressor) readInt(len int) (int, error) {
	v, err := d.xor.in.ReadInt(len)
	return int(v), err
}

// Next returns next decompressed double. Returns (0, false) when stream ends.
func (d *Decompressor) Next() (float64, bool, error) {
	// parse control bits like AbstractElfDecompressor.nextValue
	b0, err := d.readInt(1)
	if err != nil {
		return 0, false, err
	}
	var v float64
	if b0 == 0 {
		// case 0
		vv, ok, err := d.recoverVByBetaStar()
		if err != nil {
			return 0, false, err
		}
		if !ok {
			return 0, false, nil
		}
		v = vv
	} else if b1, err := d.readInt(1); err != nil {
		return 0, false, err
	} else if b1 == 0 {
		// case 10
		val, err := d.xor.ReadValue()
		if err != nil {
			return 0, false, err
		}
		if val == nil {
			return 0, false, nil
		}
		v = *val
	} else {
		// case 11
		bs, err := d.readInt(4)
		if err != nil {
			return 0, false, err
		}
		d.lastBetaStar = bs
		vv, ok, err := d.recoverVByBetaStar()
		if err != nil {
			return 0, false, err
		}
		if !ok {
			return 0, false, nil
		}
		v = vv
	}
	return v, true, nil
}

func (d *Decompressor) recoverVByBetaStar() (float64, bool, error) {
	val, err := d.xor.ReadValue()
	if err != nil {
		return 0, false, err
	}
	if val == nil {
		return 0, false, nil
	}
	vPrime := *val
	sp := GetSP(math.Abs(vPrime))
	if d.lastBetaStar == 0 {
		v := Get10iN(-sp - 1)
		if vPrime < 0 {
			v = -v
		}
		return v, true, nil
	}
	alpha := d.lastBetaStar - sp - 1
	v := RoundUp(vPrime, alpha)
	return v, true, nil
}
