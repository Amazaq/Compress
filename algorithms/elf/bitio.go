package elf

import "errors"

// BitWriter writes bits MSB-first into an internal byte buffer.
type BitWriter struct {
	buf    []byte
	cur    byte
	bits   int // number of bits filled in cur [0..7]
	closed bool
}

func NewBitWriter(capacity int) *BitWriter {
	return &BitWriter{buf: make([]byte, 0, capacity)}
}

func (w *BitWriter) WriteBit(bit bool) {
	if w.closed {
		return
	}
	if bit {
		w.cur |= 1 << (7 - w.bits)
	}
	w.bits++
	if w.bits == 8 {
		w.buf = append(w.buf, w.cur)
		w.cur = 0
		w.bits = 0
	}
}

// WriteInt writes len bits of n (MSB-first).
func (w *BitWriter) WriteInt(n uint64, length int) {
	for i := length - 1; i >= 0; i-- {
		w.WriteBit(((n >> uint(i)) & 1) == 1)
	}
}

// WriteLong is an alias for WriteInt for readability.
func (w *BitWriter) WriteLong(n uint64, length int) { w.WriteInt(n, length) }

func (w *BitWriter) Flush() {
	if w.closed {
		return
	}
	if w.bits > 0 {
		w.buf = append(w.buf, w.cur)
		w.cur = 0
		w.bits = 0
	}
	w.closed = true
}

func (w *BitWriter) Bytes() []byte { return w.buf }

// BitReader reads bits MSB-first from a byte slice.
type BitReader struct {
	buf  []byte
	i    int
	bits int
	cur  byte
	eof  bool
}

func NewBitReader(bs []byte) *BitReader {
	r := &BitReader{buf: bs}
	if len(bs) > 0 {
		r.cur = bs[0]
	}
	return r
}

func (r *BitReader) ReadBit() (bool, error) {
	if r.eof {
		return false, errors.New("EOF")
	}
	bit := ((r.cur >> (7 - r.bits)) & 1) == 1
	r.bits++
	if r.bits == 8 {
		r.i++
		if r.i >= len(r.buf) {
			r.eof = true
		} else {
			r.cur = r.buf[r.i]
		}
		r.bits = 0
	}
	return bit, nil
}

func (r *BitReader) ReadInt(length int) (uint64, error) {
	var v uint64
	for i := 0; i < length; i++ {
		b, err := r.ReadBit()
		if err != nil {
			return 0, err
		}
		v = (v << 1) | boolToU(b)
	}
	return v, nil
}

func (r *BitReader) ReadLong(length int) (uint64, error) { return r.ReadInt(length) }

func boolToU(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}
