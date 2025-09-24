package elf

func CompressFloat(dst []byte, src []float64) []byte {
	if len(src) == 0 {
		return dst
	}

	c := NewCompressor()
	for _, v := range src {
		c.Add(v)
	}
	c.Close()

	out := c.Bytes()
	if dst == nil {
		return out
	}
	return append(dst, out...)
}

func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	if len(src) == 0 {
		return []float64{}, nil
	}

	d := NewDecompressor(src)
	res := dst
	for {
		v, ok, err := d.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		res = append(res, v)
	}
	return res, nil
}
