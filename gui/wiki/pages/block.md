# Blocks

A block is the fundamental unit of a blockchain. It stores valuable data, in our case transactions, together with metadata that links it to the rest of the chain and proves that work was spent to create it.

{{image:block_overview.png|Each header stores the hash of the previous header and the Merkle root of its transactions (Bitcoin Developer Guide)}}

## The structure

```go
type Block struct {
    Timestamp     int64
    Transactions  []*Transaction
    PrevBlockHash []byte
    Hash          []byte
    Nonce         int
    Bits          int
}
```

- **Timestamp** is the Unix time when the block was created.
- **Transactions** is the payload. The first transaction is always the coinbase that pays the miner.
- **PrevBlockHash** is the hash of the previous block. This single field is what turns separate blocks into a chain.
- **Hash** is the SHA-256 hash of the block header. It identifies the block and is used as its key in the database.
- **Nonce** and **Bits** belong to Proof-of-Work. Bits is the difficulty and Nonce is the counter that made the hash valid.

In Bitcoin these header fields live in a separate structure and the hash is computed only from the header. Here they are kept together for simplicity, but the hash is still computed only from the header fields, never from the transactions directly.

## Creating a block

A block is never assembled and hashed in one step. Its hash is the result of mining:

```go
func NewBlockContext(ctx context.Context, transactions []*Transaction, prevBlockHash []byte, targetBits int, progress *MiningProgress) (*Block, error) {
    block := &Block{
        Timestamp:     time.Now().Unix(),
        Transactions:  transactions,
        PrevBlockHash: prevBlockHash,
        Hash:          []byte{},
        Nonce:         0,
        Bits:          targetBits,
    }

    pow := NewProofOfWork(block)
    nonce, hash, err := pow.Run(ctx, progress)
    if err != nil {
        return nil, err
    }

    block.Hash = hash
    block.Nonce = nonce

    return block, nil
}
```

The context allows the Mining screen to abort a long search, and the progress object feeds the live nonce counter you see there. `NewBlock` is the short version without cancellation that the P2P simulation uses.

## Transactions inside the header

The header cannot contain every transaction, so it contains one hash that represents all of them:

```go
func (b *Block) HashTransactions() []byte {
    var transactions [][]byte

    for _, tx := range b.Transactions {
        transactions = append(transactions, tx.Serialize())
    }
    mTree := NewMerkleTree(transactions)
    if mTree.RootNode == nil {
        return nil
    }

    return mTree.RootNode.Data
}
```

This is the Merkle root. Change a single byte in any transaction and the root changes, the header changes and the block hash is no longer valid. The Merkle tree chapter explains why a tree is used instead of hashing everything at once.

## The genesis block

The first block has no predecessor, so its `PrevBlockHash` is empty:

```go
func NewGenesisBlock(coinbase *Transaction, targetBits int) (*Block, error) {
    return NewBlock([]*Transaction{coinbase}, []byte{}, targetBits)
}
```

Its coinbase carries a message instead of a signature. Bitcoin used a newspaper headline, this project uses a movie-like quote about September 30th, 1998. You can read it in the inputs of block #0 on the Blockchain screen.

## Try it

Open the Blockchain screen, click any block and compare its **Previous** field with the hash of the block to the left. Then look at **Merkle root**, **Nonce** and **Target** and keep them in mind for the next chapter.
