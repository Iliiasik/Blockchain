# Proof-of-Work

Adding a block to the chain must be hard, otherwise anyone could rewrite history in seconds. Proof-of-Work makes the creator of a block perform a large amount of computation that everybody else can verify instantly.

{{image:target_nbits.png|Bitcoin packs the target into nBits, this project uses 2^(256 - bits) directly (Bitcoin Developer Guide)}}

## The idea

A block is valid only if its hash, read as a 256-bit number, is lower than a **target**. SHA-256 output is unpredictable, so the only way to find such a hash is to try different inputs. The input that changes between attempts is the **nonce**.

The difficulty is expressed in bits. The target is `1` shifted left by `256 - bits`:

```go
func NewProofOfWork(b *Block) *ProofOfWork {
    target := big.NewInt(1)
    target.Lsh(target, uint(256-b.Bits))

    pow := &ProofOfWork{b, target}
    return pow
}
```

With 16 bits the target is `0001000...0` in hex. A hash is below it only when its first 16 bits are zero. Every extra bit halves the number of valid hashes, so the expected number of attempts is `2^bits`: about 65 thousand for 16 bits, 1 million for 20, 268 million for 28.

## What is hashed

```go
func (pow *ProofOfWork) prepareHeader() []byte {
    return bytes.Join(
        [][]byte{
            pow.block.PrevBlockHash,
            pow.block.HashTransactions(),
            utils.IntToHex(pow.block.Timestamp),
            utils.IntToHex(int64(pow.block.Bits)),
        },
        []byte{},
    )
}
```

The previous hash, the Merkle root, the timestamp and the difficulty form the header. The nonce is appended as 8 big-endian bytes. Because the previous hash is part of the header, changing any old block changes its hash and invalidates the Proof-of-Work of every block after it.

## The mining loop

```go
for nonce := 0; nonce < maxNonce; nonce++ {
    binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
    hash := sha256.Sum256(data)

    if hasLeadingZeroBits(hash, pow.block.Bits) {
        progress.report(nonce, hash)
        return nonce, hash[:], nil
    }

    if nonce%progressInterval == 0 {
        progress.report(nonce, hash)
        if ctx.Err() != nil {
            return 0, nil, ErrMiningCancelled
        }
    }
}
```

Two details make this loop fast. The header is prepared **once**, before the loop, so the Merkle tree is not rebuilt millions of times. Only the last 8 bytes of the buffer are rewritten for every nonce. The comparison with the target is done directly on the bytes:

```go
func hasLeadingZeroBits(hash [32]byte, bits int) bool {
    full := bits / 8
    for i := 0; i < full; i++ {
        if hash[i] != 0 {
            return false
        }
    }

    rest := bits % 8
    return rest == 0 || hash[full]>>(8-rest) == 0
}
```

"Hash is lower than 2^(256 - bits)" means exactly "the first bits of the hash are zero", so no big integer arithmetic is needed in the hot path. The Mining screen shows the resulting hash rate.

## Validation

Mining is slow, checking is a single hash:

```go
func (pow *ProofOfWork) Validate() bool {
    var hashInt big.Int

    data := pow.prepareData(pow.block.Nonce)
    hash := sha256.Sum256(data)
    hashInt.SetBytes(hash[:])

    return hashInt.Cmp(pow.target) == -1
}
```

This asymmetry is the whole point. The block inspector runs `Validate` for every block you open and shows the **PoW valid** badge. The P2P simulation runs it on every block before a node accepts a foreign chain.

## Where the idea comes from

Proof-of-Work is older than Bitcoin. Adam Back's Hashcash (1997) asked the sender of an email to find a hash with a number of leading zero bits. For one message that costs a fraction of a second, for a spammer sending millions it becomes expensive. Bitcoin reuses the same puzzle with SHA-256 and turns it into a vote: one unit of computation is one unit of influence over which chain is accepted.

Two properties make it work:

- **Hard to produce.** There is no shortcut. The only way to a valid hash is to try nonces one by one.
- **Easy to check.** Anyone verifies a block with a single hash, the `Validate` call shown above.

## Why history is expensive to rewrite

The hash of every block is part of the header of the next one. To change a transaction in block #5 an attacker has to mine block #5 again, then #6, #7 and every block after it, and do it faster than the rest of the network keeps extending the honest chain. Nodes follow the chain with the most work, so the attacker only wins with more than half of all hashing power. This is the **51% attack**. Every new block on top of a transaction makes rewriting it more expensive, which is why Bitcoin users usually wait for about six confirmations before treating a payment as final.

## Difficulty in real networks

Bitcoin retargets the difficulty every 2016 blocks so that a block appears roughly every ten minutes, no matter how much hashing power joins the network. This project keeps the difficulty manual: every block stores its own `Bits`, and you choose the value on the Mining screen to feel how quickly the work grows.

## Try it

Mine a block at 12 bits, then at 20 bits. Watch the nonce counter, the hash rate and the **Progress** tile which compares the attempts made so far with the expected `2^bits`. Sometimes you will be lucky and finish at 10 percent, sometimes it will take 300 percent. Mining is a lottery where every ticket costs one hash.
