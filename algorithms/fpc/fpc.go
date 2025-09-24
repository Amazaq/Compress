package fpc

import (
	"math"
	"math/bits"
	"myalgo/common"
)

func Compress(dst []byte, src []uint64) []byte {
	bs := &common.ByteWrapper{Stream: &dst, Count: 0}
	bs.AppendBits(uint64(len(src)), 14)
	dfcm := NewDfcmPredictor(16)
	fcm := NewFcmPredictor(16)
	var dtmp, ftmp, cnt, xor uint64
	for _, u := range src {
		dtmp = dfcm.PredictNext() ^ u
		ftmp = fcm.PredictNext() ^ u
		if bits.LeadingZeros64(dtmp)/8 > bits.LeadingZeros64(ftmp)/8 {
			cnt = uint64(bits.LeadingZeros64(dtmp) / 8)
			if cnt == 8 {
				cnt -= 1
			}
			xor = dtmp
			bs.AppendBits(cnt|0b1000, 4)
		} else {
			cnt = uint64(bits.LeadingZeros64(ftmp) / 8)
			if cnt == 8 {
				cnt -= 1
			}
			xor = ftmp
			bs.AppendBits(cnt|0b0000, 4)
		}
		bs.AppendBits(xor, int((8-cnt)*8))
		dfcm.Update(u)
		fcm.Update(u)
	}
	return dst
}

func Decompress(dst []uint64, src []byte) ([]uint64, error) {
	bs := &common.ByteWrapper{Stream: &src, Count: 8}
	size, err := bs.ReadBits(14)
	if err != nil {
		return nil, err
	}
	dfcm := NewDfcmPredictor(16)
	fcm := NewFcmPredictor(16)
	var pred, xor, cnt uint64
	for i := uint64(0); i < size; i++ {
		if bit, err := bs.ReadBit(); err != nil {
			return nil, err
		} else {
			cnt, err = bs.ReadBits(3)
			if err != nil {
				return nil, err
			}
			if bit {
				pred = dfcm.PredictNext()
			} else {
				pred = fcm.PredictNext()
			}
			xor, err = bs.ReadBits(int(8 * (8 - cnt)))
			if err != nil {
				return nil, err
			}
			xor ^= pred
			dst = append(dst, xor)
			dfcm.Update(xor)
			fcm.Update(xor)
		}
	}
	return dst, nil
}
func CompressFloat(dst []byte, src []float64) []byte {
	bs := &common.ByteWrapper{Stream: &dst, Count: 0}
	bs.AppendBits(uint64(len(src)), 64)
	dfcm := NewDfcmPredictor(16)
	fcm := NewFcmPredictor(16)
	var dtmp, ftmp, cnt, xor uint64
	for _, u := range src {
		uint64U := math.Float64bits(u)
		dtmp = dfcm.PredictNext() ^ uint64U
		ftmp = fcm.PredictNext() ^ uint64U
		if bits.LeadingZeros64(dtmp)/8 > bits.LeadingZeros64(ftmp)/8 {
			cnt = uint64(bits.LeadingZeros64(dtmp) / 8)
			if cnt == 8 {
				cnt -= 1
			}
			xor = dtmp
			bs.AppendBits(cnt|0b1000, 4)
		} else {
			cnt = uint64(bits.LeadingZeros64(ftmp) / 8)
			if cnt == 8 {
				cnt -= 1
			}
			xor = ftmp
			bs.AppendBits(cnt|0b0000, 4)
		}
		bs.AppendBits(xor, int((8-cnt)*8))
		dfcm.Update(uint64U)
		fcm.Update(uint64U)
	}
	return dst
}

func DecompressFloat(dst []float64, src []byte) ([]float64, error) {
	bs := &common.ByteWrapper{Stream: &src, Count: 8}
	size, err := bs.ReadBits(64)
	if err != nil {
		return nil, err
	}
	dfcm := NewDfcmPredictor(16)
	fcm := NewFcmPredictor(16)
	var pred, xor, cnt uint64
	for i := uint64(0); i < size; i++ {
		if bit, err := bs.ReadBit(); err != nil {
			return nil, err
		} else {
			cnt, err = bs.ReadBits(3)
			if err != nil {
				return nil, err
			}
			if bit {
				pred = dfcm.PredictNext()
			} else {
				pred = fcm.PredictNext()
			}
			xor, err = bs.ReadBits(int(8 * (8 - cnt)))
			if err != nil {
				return nil, err
			}
			xor ^= pred
			dst = append(dst, math.Float64frombits(xor))
			dfcm.Update(xor)
			fcm.Update(xor)
		}
	}
	return dst, nil
}
