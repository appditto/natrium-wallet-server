package utils

import (
	"errors"
	"fmt"
	"math"
	"math/big"
)

const rawPerNanoStr = "1000000000000000000000000000000"
const rawPerBananoStr = "100000000000000000000000000000"

var rawPerBanano, _ = new(big.Float).SetString(rawPerBananoStr)
var rawPerNano, _ = new(big.Float).SetString(rawPerNanoStr)

const bananoPrecision = 100   // 0.01 BANANO precision
const nanoPrecision = 1000000 // 0.000001 NANO precision

// Raw to Big - converts raw amount to a big.Int
func RawToBigInt(raw string) (*big.Int, error) {
	rawBig, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Unable to convert %s to big int", raw))
	}
	return rawBig, nil
}

// RawToBanano - Converts Raw amount to usable Banano amount
func RawToBanano(raw string, truncate bool) (float64, error) {
	rawBig, ok := new(big.Float).SetString(raw)
	if !ok {
		err := errors.New(fmt.Sprintf("Unable to convert %s to int", raw))
		return -1, err
	}
	asBanano := rawBig.Quo(rawBig, rawPerBanano)
	f, _ := asBanano.Float64()
	if !truncate {
		return f, nil
	}

	return math.Trunc(f/0.01) * 0.01, nil
}

// RawToNano - Converts Raw amount to usable Nano amount
func RawToNano(raw string, truncate bool) (float64, error) {
	rawBig, ok := new(big.Float).SetString(raw)
	if !ok {
		err := errors.New(fmt.Sprintf("Unable to convert %s to int", raw))
		return -1, err
	}
	asNano := rawBig.Quo(rawBig, rawPerNano)
	if !truncate {
		f, _ := asNano.Float64()
		return f, nil
	}
	// Truncate precision beyond 0.000001
	bf := big.NewFloat(0).SetPrec(1000000).Set(asNano)
	bu := big.NewFloat(0).SetPrec(1000000).SetFloat64(0.000001)

	bf.Quo(bf, bu)

	// Truncate:
	i := big.NewInt(0)
	bf.Int(i)
	bf.SetInt(i)

	f, _ := bf.Mul(bf, bu).Float64()
	return f, nil
}

// BananoToRaw - Converts Banano amount to Raw amount
func BananoToRaw(banano float64) string {
	bananoInt := int(banano * 100)
	// 0.01 banano
	bananoRaw, _ := new(big.Int).SetString("1000000000000000000000000000", 10)

	res := bananoRaw.Mul(bananoRaw, big.NewInt(int64(bananoInt)))

	return fmt.Sprintf("%d", res)
}

// NanoToRaw - Converts Nano amount to Raw amount
func NanoToRaw(nano float64) string {
	nanoInt := int(nano * 1000000)
	// 0.000001 nano
	nanoRaw, _ := new(big.Int).SetString("1000000000000000000000000", 10)

	res := nanoRaw.Mul(nanoRaw, big.NewInt(int64(nanoInt)))

	return fmt.Sprintf("%d", res)
}
