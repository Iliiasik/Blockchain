# Base58Check

Base58 is a binary-to-text encoding designed for values that humans read and type. It uses 58 alphanumeric characters and deliberately omits the ones that are easy to confuse:

- `0` (zero) and `O` (uppercase o)
- `l` (lowercase L) and `I` (uppercase i)
- `+` and `/`, which also break double click selection

{{image:p2pkh_create.png|Creating a public key hash for a payment, Base58Check turns that hash into the address you copy (Bitcoin Developer Guide)}}

## The algorithm

Base58 treats the input bytes as one large number and repeatedly divides it by 58. Each remainder selects one character of the alphabet:

```go
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
```

Remainders come out from the least significant digit, so the result is reversed at the end.

## Leading zeros

Leading zero bytes disappear when bytes are turned into a number. To keep them, every leading `0x00` byte is written as the first alphabet character, `1`. This is why every address in this application starts with `1`: the version byte is `0x00`.

Decoding does the reverse: it counts the leading `1` characters, converts the rest back into a number and restores the zero bytes.

```go
for _, b := range input {
    if b != b58Alphabet[0] {
        break
    }
    zeroBytes++
}
```

Characters that are not in the alphabet make `Base58Decode` return `nil`, so invalid input is rejected instead of silently producing a wrong key hash.

## The "Check" part

Base58Check adds a 4 byte checksum before encoding, as described in the Wallets chapter. The checksum catches typos, and the version byte tells what kind of data the string contains.

## Summary

- **Compact.** 25 bytes become 33 or 34 characters.
- **Readable.** No ambiguous characters and no punctuation.
- **Safe.** Combined with the checksum, a typo is detected before any coins are sent.
