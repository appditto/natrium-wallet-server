package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRawToBigInt(t *testing.T) {
	raw := "1000000000000000000000000000000"

	asInt, err := RawToBigInt(raw)
	assert.Equal(t, nil, err)
	assert.Equal(t, "1000000000000000000000000000000", asInt.String())
}

func TestRawToBanano(t *testing.T) {
	raw := "1000000000000000000000000000000"

	converted, err := RawToBanano(raw, true)
	assert.Equal(t, nil, err)
	assert.Equal(t, 10.0, converted)

	raw = "5673500000000000000000000000000"
	converted, err = RawToBanano(raw, true)
	assert.Equal(t, nil, err)
	assert.Equal(t, 56.73, converted)

}

func TestRawToNano(t *testing.T) {
	raw := "1000000000000000000000000000000"

	converted, err := RawToNano(raw, true)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1.0, converted)

	raw = "5673567900000000000000000000000"
	converted, err = RawToNano(raw, true)
	assert.Equal(t, nil, err)
	assert.Equal(t, 5.673568, converted)
}

func TestBananoToRaw(t *testing.T) {
	amount := 10.0

	converted := BananoToRaw(amount)
	assert.Equal(t, "1000000000000000000000000000000", converted)
}

func TestNanoToRaw(t *testing.T) {
	amount := 1.0

	converted := NanoToRaw(amount)
	assert.Equal(t, "1000000000000000000000000000000", converted)
}
