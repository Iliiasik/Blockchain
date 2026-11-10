# Wallets and addresses

A wallet does not store coins. Coins live in transaction outputs on the chain. A wallet stores the **private key** that can unlock those outputs and derives a human friendly **address** from the public key.

{{image:p2pkh_unlock.png|The signature and the full public key must match the public key hash stored in the output (Bitcoin Developer Guide)}}

## Key pair

```go
func newKeyPair() (ecdsa.PrivateKey, []byte) {
    curve := elliptic.P256()
    private, err := ecdsa.GenerateKey(curve, rand.Reader)
    if err != nil {
        log.Panic(err)
    }
    pubKey := make([]byte, 2*coordinateSize)
    private.PublicKey.X.FillBytes(pubKey[:coordinateSize])
    private.PublicKey.Y.FillBytes(pubKey[coordinateSize:])

    return *private, pubKey
}
```

The private key is a random number. The public key is a point on the P-256 elliptic curve, stored as its X and Y coordinates, 32 bytes each. Bitcoin uses the secp256k1 curve instead, but the idea is identical: computing the public key from the private one is easy, the opposite is practically impossible.

## Public key cryptography and signatures

A key pair splits two jobs. The **private key** is a secret known only to its owner, the **public key** can be shared with anyone. Data signed with the private key can be checked by anyone who has the public key, and nobody can produce a valid signature without the private key.

In a blockchain this gives three guarantees at once:

1. **Ownership.** Only the holder of the private key can spend an output locked to the matching public key hash.
2. **Integrity.** The signature covers the transaction data. Change one byte, for example the amount, and verification fails.
3. **Non-repudiation.** A signed transaction can not be denied later by its sender.

The private key never leaves `wallet.dat`. A transaction carries only the public key and the signature, as the Transactions chapter shows. ECDSA also mixes a random number into every signature, so signing the same data twice gives two different signatures, and both are valid.

## From public key to address

```go
func HashPubKey(pubKey []byte) []byte {
    publicSHA256 := sha256.Sum256(pubKey)

    RIPEMD160Hasher := ripemd160.New()
    _, _ = RIPEMD160Hasher.Write(publicSHA256[:])

    return RIPEMD160Hasher.Sum(nil)
}

func PubKeyHashToAddress(pubKeyHash []byte) string {
    versionedPayload := append([]byte{version}, pubKeyHash...)
    fullPayload := append(versionedPayload, checksum(versionedPayload)...)

    return string(Base58Encode(fullPayload))
}
```

1. Hash the public key with SHA-256 and then RIPEMD-160. The result is a 20 byte **public key hash**. Outputs are locked to this value.
2. Prepend the version byte `0x00`.
3. Append a 4 byte checksum, the first bytes of a double SHA-256 of the versioned payload.
4. Encode the 25 bytes with Base58.

The Wallets screen shows all three parts of every address: version, public key hash and checksum.

## Validation

```go
func ValidateAddress(address string) bool {
    pubKeyHash := Base58Decode([]byte(address))

    if len(pubKeyHash) != 1+ripemd160.Size+addressChecksumLen || pubKeyHash[0] != version {
        return false
    }

    actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
    targetChecksum := checksum(pubKeyHash[:len(pubKeyHash)-addressChecksumLen])

    return bytes.Equal(actualChecksum, targetChecksum)
}
```

A single mistyped character changes the checksum, so the transaction form refuses the address before any coins are sent.

## Storage

All wallets are kept in `wallet.dat`. `Wallet` implements `GobEncode` and `GobDecode` so the elliptic curve, an interface value, does not have to be serialized. The file is written with `0600` permissions, readable only by your user, because anyone who copies it can spend your coins.

## Types of wallets

- **Hot wallets** are connected to the internet, convenient but more exposed.
- **Cold wallets** keep keys offline for long term storage.
- **Hardware wallets** are devices that sign transactions without revealing the key.
- **Software wallets** are programs on a computer or phone.

The wallet in this application is a **hot, non-custodial software wallet**: keys are generated and kept locally, nobody else holds them.

## Try it

Create two wallets, copy an address, change one character and paste it into the recipient field of the Transactions screen. The field turns invalid because the checksum no longer matches.
