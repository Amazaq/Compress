package elf

import "math"

var fAlpha = [...]int{0, 4, 7, 10, 14, 17, 20, 24, 27, 30, 34, 37, 40, 44, 47, 50, 54, 57, 60, 64, 67}

var map10iP = [...]float64{1.0, 1.0e1, 1.0e2, 1.0e3, 1.0e4, 1.0e5, 1.0e6, 1.0e7, 1.0e8, 1.0e9, 1.0e10, 1.0e11, 1.0e12, 1.0e13, 1.0e14, 1.0e15, 1.0e16, 1.0e17, 1.0e18, 1.0e19, 1.0e20}

var map10iN = [...]float64{1.0, 1.0e-1, 1.0e-2, 1.0e-3, 1.0e-4, 1.0e-5, 1.0e-6, 1.0e-7, 1.0e-8, 1.0e-9, 1.0e-10, 1.0e-11, 1.0e-12, 1.0e-13, 1.0e-14, 1.0e-15, 1.0e-16, 1.0e-17, 1.0e-18, 1.0e-19, 1.0e-20}

var mapSPGreater1 = [...]float64{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000}

var mapSPLess1 = [...]float64{1, 0.1, 0.01, 0.001, 0.0001, 0.00001, 0.000001, 0.0000001, 0.00000001, 0.000000001, 0.0000000001}

const log2_10 = 3.3219280948873626 // math.Log2(10)

func get10iP(i int) float64 {
	if i < 0 {
		panic("i must be >= 0")
	}
	if i >= len(map10iP) {
		return parsePow10Pos(i)
	}
	return map10iP[i]
}

func Get10iN(i int) float64 {
	if i < 0 {
		panic("i must be >= 0")
	}
	if i >= len(map10iN) {
		return parsePow10Neg(i)
	}
	return map10iN[i]
}

func parsePow10Pos(i int) float64 { return math.Pow(10, float64(i)) }
func parsePow10Neg(i int) float64 { return math.Pow(10, float64(-i)) }

func GetFAlpha(alpha int) int {
	if alpha < 0 {
		panic("alpha must be >= 0")
	}
	if alpha >= len(fAlpha) {
		return int(math.Ceil(float64(alpha) * (math.Log(10) / math.Log(2))))
	}
	return fAlpha[alpha]
}

// GetSP returns sp similar to Elf64Utils.getSP
func GetSP(v float64) int {
	if v >= 1 {
		for i := 0; i < len(mapSPGreater1)-1; i++ {
			if v < mapSPGreater1[i+1] {
				return i
			}
		}
	} else {
		for i := 1; i < len(mapSPLess1); i++ {
			if v >= mapSPLess1[i] {
				return -i
			}
		}
	}
	return int(math.Floor(math.Log10(v)))
}

// getSPAnd10iNFlag replicates Elf64Utils private logic, returns [sp, flag]
func getSPAnd10iNFlag(v float64) (int, int) {
	if v >= 1 {
		for i := 0; i < len(mapSPGreater1)-1; i++ {
			if v < mapSPGreater1[i+1] {
				return i, 0
			}
		}
	} else {
		for i := 1; i < len(mapSPLess1); i++ {
			if v >= mapSPLess1[i] {
				flag := 0
				if v == mapSPLess1[i] {
					flag = 1
				}
				return -i, flag
			}
		}
	}
	log10v := math.Log10(v)
	sp := int(math.Floor(log10v))
	flag := 0
	if float64(int64(log10v)) == log10v {
		flag = 1
	}
	return sp, flag
}

func getSignificantCount(v float64, sp int, lastBetaStar int) int {
	var i int
	if lastBetaStar != math.MaxInt32 && lastBetaStar != 0 {
		i = max(lastBetaStar-sp-1, 1)
	} else if lastBetaStar == math.MaxInt32 {
		i = 17 - sp - 1
	} else if sp >= 0 {
		i = 1
	} else {
		i = -sp
	}

	temp := v * get10iP(i)
	tempLong := float64(int64(temp))
	for tempLong != temp {
		i++
		temp = v * get10iP(i)
		tempLong = float64(int64(temp))
	}

	if temp/get10iP(i) != v {
		return 17
	}
	for i > 0 && int64(tempLong)%10 == 0 {
		i--
		tempLong = float64(int64(tempLong) / 10)
	}
	return sp + i + 1
}

// GetAlphaAndBetaStar replicates Elf64Utils.getAlphaAndBetaStar
func GetAlphaAndBetaStar(v float64, lastBetaStar int) (alpha int, betaStar int) {
	if v < 0 {
		v = -v
	}
	sp, flag := getSPAnd10iNFlag(v)
	beta := getSignificantCount(v, sp, lastBetaStar)
	alpha = beta - sp - 1
	if flag == 1 {
		betaStar = 0
	} else {
		betaStar = beta
	}
	return
}

// RoundUp replicates Elf64Utils.roundUp
func RoundUp(v float64, alpha int) float64 {
	scale := get10iP(alpha)
	if v < 0 {
		return math.Floor(v*scale) / scale
	}
	return math.Ceil(v*scale) / scale
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
