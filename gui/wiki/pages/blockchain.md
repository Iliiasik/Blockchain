# Blockchain and storage

A blockchain is an ordered, back-linked list of blocks. Each block points to its predecessor through `PrevBlockHash`, so the newest block, the **tip**, is enough to reach every other block.

{{image:height_depth.png|Height counts up from the genesis block, depth counts back from the tip (Bitcoin Developer Guide)}}

## The structure

```go
type Blockchain struct {
    tip     []byte
    Db      *bolt.DB
    Mempool *Mempool
}
```

The struct keeps only the hash of the tip, a handle to the database and the mempool. Blocks themselves stay on disk and are loaded only when needed.

## Why BoltDB

The application uses BoltDB (the maintained `go.etcd.io/bbolt` fork). It is a pure Go key/value store that keeps everything in one file, `blockchain.db`, and supports concurrent readers with a single writer. Bitcoin Core uses LevelDB in a similar way.

Two buckets are used:

- **blocks** maps a block hash to the serialized block. The special key `l` stores the hash of the last block.
- **chainstate** stores the UTXO set, explained in its own chapter.

## How Bitcoin Core does it

Older versions of Bitcoin Core, the ones the original course describes, split the data into two LevelDB databases:

- **blocks** keeps metadata about blocks: `b` plus a block hash points to a block index record, `f` plus a file number to information about a block file, `l` to the last block file in use. The blocks themselves live in separate `blk*.dat` files.
- **chainstate** keeps `c` plus a transaction hash mapped to the unspent outputs of that transaction, and `B` to the hash of the block up to which these outputs are valid.

This project follows the same split in a simpler form. Whole blocks go straight into the `blocks` bucket, only the `l` key survives from the metadata, and `chainstate` holds the UTXO set.

## Creating the chain

```go
err = db.Update(func(tx *bolt.Tx) error {
    b, err := tx.CreateBucket([]byte(blocksBucket))
    if err != nil {
        return fmt.Errorf("could not create bucket: %v", err)
    }

    if err := b.Put(genesis.Hash, genesis.Serialize()); err != nil {
        return fmt.Errorf("could not store genesis block: %v", err)
    }

    if err := b.Put([]byte("l"), genesis.Hash); err != nil {
        return fmt.Errorf("could not update last block hash: %v", err)
    }

    tip = genesis.Hash
    return nil
})
```

Everything happens inside one transaction, so the database never contains a block without the `l` pointer or the opposite. The database is opened with a one second timeout. If another copy of the application already holds the file, you get a clear error instead of a frozen window.

## Walking the chain

Blocks are read backwards with an iterator:

```go
func (i *BlockchainIterator) Next() *Block {
    var block *Block

    err := i.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte(blocksBucket))
        if b == nil {
            return nil
        }
        if encodedBlock := b.Get(i.currentHash); encodedBlock != nil {
            block = DeserializeBlock(encodedBlock)
        }

        return nil
    })

    if err != nil || block == nil {
        return nil
    }

    i.currentHash = block.PrevBlockHash

    return block
}
```

The iteration stops at the genesis block, whose previous hash is empty. The explorer, the wallet history and the UTXO reindex are all built on this loop.

## Adding a block

`MineBlock` verifies every transaction, reads the current tip, mines a block on top of it and stores the result:

```go
err = bc.Db.Update(func(tx *bolt.Tx) error {
    b := tx.Bucket([]byte(blocksBucket))
    if !bytes.Equal(b.Get([]byte("l")), lastHash) {
        return errors.New("chain tip changed while mining")
    }
    if err := b.Put(newBlock.Hash, newBlock.Serialize()); err != nil {
        return err
    }
    return b.Put([]byte("l"), newBlock.Hash)
})
```

The tip check protects the chain from a race: mining can take minutes and the block must still extend the block it was mined on. After the block is stored, its transactions are removed from the mempool and the UTXO set is updated with `UTXOSet.Update`.

## One handle for the whole application

Opening a BoltDB file is expensive and only one process may write to it. The application opens the chain once at start, shares the handle between all screens and closes it when the window closes. Every heavy read, like rebuilding balances or walking the chain, runs in a background goroutine and the result is sent to the interface with `fyne.Do`. This is why the interface stays responsive even while a block is being mined.

## Try it

Mine a few blocks and open **Reset** on the Blockchain screen. It deletes `blockchain.db` and `mempool.dat` but keeps `wallet.dat`, so you can start a new chain with the same wallets.
