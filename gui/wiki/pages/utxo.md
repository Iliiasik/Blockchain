# UTXO set

UTXO stands for **Unspent Transaction Output**. The set of all UTXOs is the real state of the system: every coin that exists right now is an output that no input has referenced yet.

{{image:utxo_bookkeeping.png|Value moves from outputs to inputs, whatever is left unspent forms the UTXO set (Bitcoin Developer Guide)}}

## The problem it solves

To find a balance you could walk the whole chain, collect every output locked to an address and subtract the ones that were spent later. That is correct, and it is exactly what `Blockchain.FindUTXO` does. But Bitcoin's chain is hundreds of gigabytes, and nobody wants to read it to show one number.

The UTXO set is a cache of the result. It lives in the `chainstate` bucket of the same BoltDB file. The key is a transaction ID, the value is the list of its outputs that are still unspent.

## Keeping output indexes stable

An input refers to an output by its index inside the transaction. If output 0 is spent and removed from the list, output 1 would move to position 0 and the next spender would reference the wrong output. To avoid this, spent outputs are replaced with an empty placeholder instead of being removed:

```go
for outIdx := range outs.Outputs {
    if outIdx == vin.Vout {
        outs.Outputs[outIdx] = TXOutput{}
    }
    if len(outs.Outputs[outIdx].PubKeyHash) > 0 {
        unspent++
    }
}

if unspent == 0 {
    err = b.Delete(vin.Txid)
} else {
    err = b.Put(vin.Txid, outs.Serialize())
}
```

When every output of a transaction is spent, the whole entry is deleted. Readers simply skip placeholders.

## Updating after a block

```go
func (u UTXOSet) Update(block *Block) error
```

For every transaction in a new block, `Update` marks the outputs referenced by its inputs as spent and adds all of its own outputs. This is cheap: it touches only the entries mentioned in one block.

## Reindexing

```go
func (u UTXOSet) Reindex() error {
    db := u.Blockchain.Db
    bucketName := []byte(utxoBucket)
    UTXO := u.Blockchain.FindUTXO()

    return db.Update(func(tx *bolt.Tx) error {
        if err := tx.DeleteBucket(bucketName); err != nil && !errors.Is(err, berrors.ErrBucketNotFound) {
            return err
        }

        b, err := tx.CreateBucket(bucketName)
        if err != nil {
            return err
        }

        for txID, outs := range UTXO {
            key, err := hex.DecodeString(txID)
            if err != nil {
                return err
            }

            if err := b.Put(key, outs.Serialize()); err != nil {
                return err
            }
        }

        return nil
    })
}
```

`Reindex` throws the cache away and rebuilds it from the chain. It runs when the genesis block is created and whenever you press **Reindex** on the UTXO screen.

## Queries

- `FindSpendableOutputs` collects outputs of one owner until the requested amount is reached. Outputs already spent by a pending transaction are skipped, so two transactions can never try to spend the same output.
- `FindUTXO` returns outputs of one owner, or all of them when the filter is empty. Each result carries its transaction ID, index and a **Pending** flag.
- `Balances` sums all outputs in one pass. The Overview uses it to compute the coin supply.
- `CountTransactions` returns the number of keys in the bucket.

## Try it

Open the UTXO screen and pick a wallet. Then send coins from it without mining. One of its outputs turns **in mempool**: it is still unspent on the chain, but already promised to a pending transaction. Mine a block and the output disappears, replaced by new outputs for the recipient and the change.
