package utils

import (
	"encoding/hex"
	"testing"

	utils "github.com/bananocoin/boompow/libs/utils/testing"
)

func TestValidateAddressNano(t *testing.T) {
	// Valid
	valid := "nano_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3r"
	utils.AssertEqual(t, true, ValidateAddress(valid, false))
	valid = "xrb_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3r"
	utils.AssertEqual(t, true, ValidateAddress(valid, false))
	// Invalid
	invalid := "nano_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3ra"
	utils.AssertEqual(t, false, ValidateAddress(invalid, false))
	invalid = "nano_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3rb"
	utils.AssertEqual(t, false, ValidateAddress(invalid, false))
	invalid = "ban_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3r"
	utils.AssertEqual(t, false, ValidateAddress(invalid, false))
}

func TestValidateAddressBanano(t *testing.T) {
	// Valid
	valid := "ban_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3r"
	utils.AssertEqual(t, true, ValidateAddress(valid, true))
	// Invalid
	invalid := "ban_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3ra"
	utils.AssertEqual(t, false, ValidateAddress(invalid, true))
	invalid = "ban_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3rb"
	utils.AssertEqual(t, false, ValidateAddress(invalid, true))
	invalid = "nano_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3r"
	utils.AssertEqual(t, false, ValidateAddress(invalid, true))
}

func TestAddressToPub(t *testing.T) {
	address := "ban_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3r"
	pub, err := AddressToPub(address)
	utils.AssertEqual(t, nil, err)
	utils.AssertEqual(t, "7fc9064e4d713af2afc73c1527334b665972eb57d65093a378a3e40dbb48ec43", hex.EncodeToString(pub))
	address = "nano_1zyb1s96twbtycqwgh1o6wsnpsksgdoohokikgjqjaz63pxnju457pz8tm3r"
	pub, err = AddressToPub(address)
	utils.AssertEqual(t, nil, err)
	utils.AssertEqual(t, "7fc9064e4d713af2afc73c1527334b665972eb57d65093a378a3e40dbb48ec43", hex.EncodeToString(pub))
}
