package utils

import (
	"encoding/base32"
	"errors"
	"regexp"

	"github.com/appditto/natrium-wallet-server/utils/ed25519"
	"golang.org/x/crypto/blake2b"
)

// nano uses a non-standard base32 character set.
const EncodeNano = "13456789abcdefghijkmnopqrstuwxyz"

var NanoEncoding = base32.NewEncoding(EncodeNano)

const bananoRegexStr = "(?:ban)(?:_)(?:1|3)(?:[13456789abcdefghijkmnopqrstuwxyz]{59})"
const nanoRegexStr = "(?:xrb|nano)(?:_)(?:1|3)(?:[13456789abcdefghijkmnopqrstuwxyz]{59})"

var bananoRegex = regexp.MustCompile(bananoRegexStr)
var nanoRegex = regexp.MustCompile(nanoRegexStr)

// ValidateAddress - Returns true if a nano/banano address is valid
func ValidateAddress(account string, bananoMode bool) bool {
	if bananoMode && !bananoRegex.MatchString(account) {
		return false
	} else if !nanoRegex.MatchString(account) {
		return false
	}

	_, err := AddressToPub(account)
	if err != nil {
		return false
	}
	return true
}

// Convert address to a public key
func AddressToPub(account string) (public_key []byte, err error) {
	address := string(account)

	if address[:4] == "xrb_" || address[:4] == "ban_" {
		address = address[4:]
	} else if address[:5] == "nano_" {
		address = address[5:]
	} else {
		return nil, errors.New("Invalid address format")
	}
	// A valid nano address is 64 bytes long
	// First 5 are simply a hard-coded string nano_ for ease of use
	// The following 52 characters form the address, and the final
	// 8 are a checksum.
	// They are base 32 encoded with a custom encoding.
	if len(address) == 60 {
		// The nano address string is 260bits which doesn't fall on a
		// byte boundary. pad with zeros to 280bits.
		// (zeros are encoded as 1 in nano's 32bit alphabet)
		key_b32nano := "1111" + address[0:52]
		input_checksum := address[52:]

		key_bytes, err := NanoEncoding.DecodeString(key_b32nano)
		if err != nil {
			return nil, err
		}
		// strip off upper 24 bits (3 bytes). 20 padding was added by us,
		// 4 is unused as account is 256 bits.
		key_bytes = key_bytes[3:]

		// nano checksum is calculated by hashing the key and reversing the bytes
		valid := NanoEncoding.EncodeToString(GetAddressChecksum(key_bytes)) == input_checksum
		if valid {
			return key_bytes, nil
		} else {
			return nil, errors.New("Invalid address checksum")
		}
	}

	return nil, errors.New("Invalid address format")
}

func GetAddressChecksum(pub ed25519.PublicKey) []byte {
	hash, err := blake2b.New(5, nil)
	if err != nil {
		panic("Unable to create hash")
	}

	hash.Write(pub)
	return Reversed(hash.Sum(nil))
}

func Reversed(str []byte) (result []byte) {
	for i := len(str) - 1; i >= 0; i-- {
		result = append(result, str[i])
	}
	return result
}
