# P2P network

A **peer-to-peer network** has no central server. Every node keeps its own copy of the chain, validates everything it receives and talks to other nodes as an equal.

{{image:chain_forks.png|Forks appear when two nodes find a block at the same height, the longer branch wins (Bitcoin Developer Guide)}}

## Why it matters

- **Redundancy.** Thousands of copies of the ledger exist, so no single failure can destroy it.
- **Resilience.** The network keeps working even if many nodes go offline.
- **No authority.** Nobody decides which transactions are valid. Every node checks the rules itself.

## Consensus: the longest valid chain

Nodes mine independently, so sometimes two of them find a block at the same height. The chain forks. Bitcoin resolves this with a simple rule: a node follows the valid chain with the most accumulated work, in practice usually the longest one. Blocks on the losing branch become **orphans** and their transactions return to the mempool.

## The simulation

The P2P screen runs three nodes in memory. They share a genesis block and each node has its own miner address. Nothing is written to disk and the main chain is not touched.

Syncing applies the longest chain rule, but only after validation:

```go
func (p *P2PDemoUI) adopt(target, source *P2PNode) syncResult {
    if target.busy || len(source.Blocks) <= len(target.Blocks) {
        return unchanged
    }
    if !validChain(source.Blocks) {
        return invalid
    }

    newBlocks := make([]*core.Block, len(source.Blocks))
    for i, block := range source.Blocks {
        newBlocks[i] = core.DeserializeBlock(block.Serialize())
    }
    target.Blocks = newBlocks

    return adopted
}
```

```go
func validChain(blocks []*core.Block) bool {
    for i, block := range blocks {
        if !core.NewProofOfWork(block).Validate() {
            return false
        }
        if i > 0 && !bytes.Equal(block.PrevBlockHash, blocks[i-1].Hash) {
            return false
        }
    }
    return true
}
```

A node checks the Proof-of-Work of every block and that every block points to the previous one. A longer chain with a single fake block is rejected. Blocks are copied through serialization, just like they would travel over the network.

## Forks

**Fork** replaces the last block of a node with a different block mined on the same parent. The node now has a chain of the same length that disagrees with the others. Blocks that are not on the longest chain are marked **FORK**. Mine one more block on the forked node and use **Auto-sync**: the longest chain wins and the other nodes switch to it.

## Node roles

Not every node does the same work:

- **Miner nodes** collect transactions from their mempool and mine new blocks. They need fast hardware and an up to date view of the tip.
- **Full nodes** download and validate every block and transaction, keep the UTXO set and relay data to their peers. They are the ones that enforce the rules.
- **SPV nodes** (simplified payment verification) keep only block headers and ask full nodes for Merkle proofs of the transactions they care about. Most wallet apps on phones work this way.

## How nodes talk

A node that joins the network has a short conversation with a peer:

1. `version` tells the peer the protocol version and the height of the local chain. If the peer's chain is longer, the node starts downloading.
2. `getblocks` asks for the blocks the node does not have yet.
3. `inv` answers with an inventory: hashes of blocks or transactions, not the data itself.
4. `getdata` requests one block or transaction by its hash.
5. `block` and `tx` carry the actual data.

When someone sends coins, their node announces the transaction with `inv`. Peers that do not know it reply with `getdata`, receive it in a `tx` message, validate it, put it into their mempool and announce it further. A miner eventually includes it in a block, and the block spreads through the same `inv`, `getdata`, `block` cycle. Every node validates the block, updates its UTXO set and drops the mined transactions from its mempool.

This application simulates three nodes inside one process to keep the focus on consensus. Each node holds its own copy of the chain, and the sync buttons stand in for the `getblocks` and `block` exchange.

## Real networks

{{image:p2p_messages.png|Data request and reply messages between Bitcoin nodes (Bitcoin Developer Guide)}}

Bitcoin nodes find each other through DNS seeds and known peers, exchange `version`, `inv`, `getdata`, `block` and `tx` messages over TCP, and relay new transactions and blocks to their neighbours. The network can be described as:

1. **Structured**, where nodes are responsible for specific data, like DHTs.
2. **Unstructured**, where peers connect randomly. Bitcoin works this way.
3. **Hybrid**, a mix of both.

## Try it

Start the simulation. Mine two blocks on Node A, sync B from A, then fork B. B now shows a FORK block. Mine on A again and press Auto-sync: B drops its fork and follows A.
