# Merkle tree

A **Merkle tree** is a binary tree of hashes built from the transactions of a block. Its root is a single 32 byte value that commits to every transaction, and it allows proving that one transaction is in a block without downloading the others.

{{image:merkle_construction.png|Bitcoin hashes every node twice, this project uses a single SHA-256 (Bitcoin Developer Guide)}}

## Building the tree

```go
func NewMerkleTree(data [][]byte) *MerkleTree {
    if len(data) == 0 {
        return &MerkleTree{}
    }

    var nodes []*MerkleNode

    for _, datum := range data {
        node := NewMerkleNode(nil, nil, datum)
        nodes = append(nodes, node)
    }

    for len(nodes) > 1 {
        if len(nodes)%2 != 0 {
            nodes = append(nodes, nodes[len(nodes)-1])
        }

        var newLevel []*MerkleNode

        for i := 0; i < len(nodes); i += 2 {
            node := NewMerkleNode(nodes[i], nodes[i+1], nil)
            newLevel = append(newLevel, node)
        }

        nodes = newLevel
    }

    return &MerkleTree{nodes[0]}
}
```

1. Every serialized transaction becomes a **leaf** containing its hash.
2. Neighbouring nodes are paired and each pair is hashed into a **parent**.
3. If a level has an odd number of nodes, the last one is duplicated.
4. The process repeats until one node is left, the **Merkle root**.

```go
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
    mNode := MerkleNode{}

    if left == nil && right == nil {
        hash := sha256.Sum256(data)
        mNode.Data = hash[:]
    } else {
        prevHashes := append(left.Data, right.Data...)
        hash := sha256.Sum256(prevHashes)
        mNode.Data = hash[:]
    }

    mNode.Left = left
    mNode.Right = right

    return &mNode
}
```

## Example

For a block with three transactions **a, b, c**:

```text
d1 = hash(a)
d2 = hash(b)
d3 = hash(c)
d4 = hash(c)            duplicate, the level is odd
d5 = hash(d1 + d2)
d6 = hash(d3 + d4)
root = hash(d5 + d6)
```

Bitcoin uses double SHA-256 for every node and writes hashes in little-endian order. This project uses a single SHA-256, which is enough to show the structure.

## Why a tree and not one hash

Hashing all transactions together would also produce one value. The tree adds **proofs**. To show that transaction `b` is in the block, you only need `d1` and `d6`:

```text
d5   = hash(d1 + hash(b))
root = hash(d5 + d6)
```

If the computed root equals the root in the header, `b` is in the block. The proof grows with the logarithm of the number of transactions: about 12 hashes for a block of 4000 transactions. This is how **Simplified Payment Verification** (SPV) clients, like mobile wallets, check payments while storing only block headers.

## In this project

`Block.HashTransactions` builds the tree and the root becomes part of the Proof-of-Work header. The block inspector shows it as **Merkle root**. Mine a block with several mempool transactions and compare its root with an empty block: the root changes with every transaction you include.
