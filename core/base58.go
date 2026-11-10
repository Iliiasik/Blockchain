package core

import (
	"Blockchain/utils"
	"bytes"
	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func Base58Encode(input []byte) []byte {
	var result []byte

	x := new(big.Int).SetBytes(input)

	base := big.NewInt(int64(len(b58Alphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}

	for _, b := range input {
		if b != 0x00 {
			break
		}
		result = append(result, b58Alphabet[0])
	}

	utils.ReverseBytes(result)

	return result
}

func Base58Decode(input []byte) []byte {
	result := big.NewInt(0)
	zeroBytes := 0

	for _, b := range input {
		if b != b58Alphabet[0] {
			break
		}
		zeroBytes++
	}

	base := big.NewInt(int64(len(b58Alphabet)))
	for _, b := range input[zeroBytes:] {
		charIndex := bytes.IndexByte(b58Alphabet, b)
		if charIndex < 0 {
			return nil
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(charIndex)))
	}

	return append(make([]byte, zeroBytes), result.Bytes()...)
}
