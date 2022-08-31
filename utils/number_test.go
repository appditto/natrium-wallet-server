package utils

import (
	"testing"
)

func TestRawToBigInt(t *testing.T) {
	raw := "1000000000000000000000000000000"

	asInt, err := RawToBigInt(raw)
	AssertEqual(t, nil, err)
	AssertEqual(t, "1000000000000000000000000000000", asInt.String())
}

func TestRawToBanano(t *testing.T) {
	raw := "1000000000000000000000000000000"

	converted, err := RawToBanano(raw, true)
	AssertEqual(t, nil, err)
	AssertEqual(t, 10.0, converted)
}

func TestRawToNano(t *testing.T) {
	raw := "1000000000000000000000000000000"

	converted, err := RawToNano(raw, true)
	AssertEqual(t, nil, err)
	AssertEqual(t, 1.0, converted)
}

func TestBananoToRaw(t *testing.T) {
	amount := 10.0

	converted := BananoToRaw(amount)
	AssertEqual(t, "1000000000000000000000000000000", converted)
}

func TestNanoToRaw(t *testing.T) {
	amount := 1.0

	converted := NanoToRaw(amount)
	AssertEqual(t, "1000000000000000000000000000000", converted)
}
