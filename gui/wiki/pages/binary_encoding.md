# Binary encoding

Hashes are computed from bytes, so every node must turn a block or a transaction into exactly the same bytes. This project uses its own small binary format instead of `encoding/gob` or JSON.

{{image:tx_structure.png|The parts of a transaction that every node must serialize the same way (Bitcoin Developer Guide)}}

## Why not gob

Gob is convenient but it writes type descriptions, can change between Go versions and does not promise a stable byte layout. If two nodes serialized the same transaction differently, they would compute different IDs and never agree. A hand written format is boring, and boring is exactly what hashing needs.

## The rules

- **Fixed order.** Fields are always written in the same sequence.
- **Length prefix.** Every byte slice is preceded by its length as an `int64`.
- **Little-endian numbers.** Integers and timestamps are written with `binary.LittleEndian`.
- **Recursion.** A block writes its transactions, a transaction writes its inputs and outputs, all with the same rules.

## Three small helpers

```go
func writeInt(buf []byte, value int64) []byte {
    return binary.LittleEndian.AppendUint64(buf, uint64(value))
}

func writeBytes(buf []byte, data []byte) []byte {
    return append(writeInt(buf, int64(len(data))), data...)
}

func readBytes(buf *bytes.Reader) []byte {
    size := readInt(buf)
    if size <= 0 || size > int64(buf.Len()) {
        return []byte{}
    }

    data := make([]byte, size)
    if _, err := io.ReadFull(buf, data); err != nil {
        return []byte{}
    }

    return data
}
```

The writers append to a byte slice and can not fail, so serialization has no error paths to forget. `readBytes` checks the declared length against the bytes that are really left. A damaged `mempool.dat` or database entry can not make the program allocate gigabytes or read past the end of the buffer.

## A transaction on the wire

```go
func (tx *Transaction) Serialize() []byte {
    buf := writeBytes(nil, tx.ID)

    buf = writeInt(buf, int64(len(tx.Vin)))
    for _, input := range tx.Vin {
        buf = writeBytes(buf, input.Txid)
        buf = writeInt(buf, int64(input.Vout))
        buf = writeBytes(buf, input.Signature)
        buf = writeBytes(buf, input.PubKey)
    }

    buf = writeInt(buf, int64(len(tx.Vout)))
    for _, output := range tx.Vout {
        buf = writeInt(buf, int64(output.Value))
        buf = writeBytes(buf, output.PubKeyHash)
    }

    return writeInt(buf, int64(tx.Fee))
}
```

The layout of a transaction with one input and two outputs looks like this:

```text
[len][ID]
[count=1]
    [len][Txid] [Vout] [len][Signature] [len][PubKey]
[count=2]
    [Value] [len][PubKeyHash]
    [Value] [len][PubKeyHash]
[Fee]
```

`DeserializeTransaction` reads the fields back in the same order. Blocks use the same approach and wrap every serialized transaction with a length prefix.

## Where the bytes are used

- **Hashing.** The transaction ID is `sha256(Serialize())` of a copy with an empty ID. The Merkle tree hashes serialized transactions.
- **Storage.** Blocks and UTXO entries are stored in BoltDB in this format. The mempool keeps one hex encoded transaction per line in `mempool.dat`.
- **Copying.** The P2P simulation copies blocks between nodes by serializing and deserializing them, the same thing a real node does when it receives a block from the network.

## Summary

Deterministic encoding gives every node identical hashes, compact storage and safe parsing. Wallets are the only data still stored with gob, because private keys never leave the machine and never need a stable hash.
