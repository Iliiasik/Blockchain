# Mempool and mining

The **mempool** (memory pool) is the waiting room for transactions. A transaction enters it right after it is signed and leaves it when a miner includes it in a block.

{{image:mining_workflow.png|Pending transactions flow from the network into a block template and then into mining (Bitcoin Developer Guide)}}

## The structure

```go
type Mempool struct {
    transactions map[string]*Transaction
    maxSize      int
    filePath     string
    mu           sync.RWMutex
    spentOutputs map[string]map[int]string
}
```

- **transactions** maps a hex transaction ID to the transaction.
- **spentOutputs** answers the question "is this output already promised to a pending transaction?" for every `(txid, index)` pair.
- **maxSize** limits the number of pending transactions. You can change it on the Transactions screen.
- **mu** makes the mempool safe to use from the mining goroutine and the interface at the same time.

## Double spend protection

```go
func (mp *Mempool) AddTransaction(tx *Transaction) error {
    mp.mu.Lock()
    defer mp.mu.Unlock()

    if len(mp.transactions) >= mp.maxSize {
        return ErrMempoolFull
    }

    txID := hex.EncodeToString(tx.ID)
    for _, input := range tx.Vin {
        inputTxID := hex.EncodeToString(input.Txid)
        if spendingTxID, exists := mp.spentOutputs[inputTxID][input.Vout]; exists && spendingTxID != txID {
            return ErrOutputReserved
        }
    }

    mp.addTransactionInternal(tx)

    return mp.saveLocked()
}
```

A transaction that tries to spend an output already used by another pending transaction is rejected with `ErrOutputReserved`, and a failed write of `mempool.dat` is reported instead of being lost. On top of that, the wallet logic allows only one pending transaction per sender, which keeps change outputs simple.

## Persistence

The mempool is saved to `mempool.dat`, one hex encoded transaction per line, every time it changes. When the application starts, it loads the file and removes every transaction whose inputs are no longer unspent or whose signature fails, for example after the chain was reset.

## Choosing transactions to mine

`GetTransactions` returns pending transactions sorted by fee, highest first, the same priority a profit seeking miner uses. On the Mining screen you choose which ones to include:

```go
fees := 0
for _, tx := range txs {
    fees += tx.Fee
}

coinbase := core.NewCoinbaseTX(miner, "", s.Subsidy+fees)
transactions := append([]*core.Transaction{coinbase}, txs...)
```

The coinbase pays the miner the **subsidy** plus every **fee**. Fees are why miners have a reason to include your transaction, and why a higher fee usually means faster confirmation.

## What happens after mining

1. `MineBlock` verifies every selected transaction.
2. Proof-of-Work finds a nonce.
3. The block is stored and becomes the new tip.
4. The included transactions are removed from the mempool.
5. `UTXOSet.Update` spends their inputs and adds their outputs.

## Try it

Lower the mempool capacity to 2 and send three transactions from three wallets. The third one is rejected because the mempool is full. Mine a block with only the cheapest transaction and watch the others wait for the next block.
