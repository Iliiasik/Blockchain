# UTXO in blockchain
UTXO (Unspent Transaction Output) represents the unspent outputs of previous transactions 
in a blockchain. Each UTXO can be used as an input in a new transaction, effectively tracking
available balances without storing account states. This model underpins the Bitcoin transaction
system and other UTXO-based cryptocurrencies.
---
{{image:utxo.png}}
---
## Usage in blockchain
In blockchain systems, UTXOs are essential for managing spendable funds and ensuring 
transaction validity:
- **Tracking spendable outputs** — only UTXOs can be used as inputs for new transactions.
- **Preventing double-spending** — once a UTXO is spent, it is removed from the set.
- **Efficient balance calculation** — wallets scan the UTXO set to compute available balances.
- **Transaction verification** — nodes validate transactions by checking referenced UTXOs and signatures.
A UTXO set is typically maintained in a database (e.g., BoltDB, LevelDB) for fast retrieval. 
When a new block is added, the UTXO set is updated: spent outputs are removed, and newly 
created outputs are added.
## Reindexing and querying
- **Reindexing** — rebuilds the UTXO set from the blockchain to ensure consistency.
- **Finding spendable outputs** — identifies UTXOs that satisfy a given amount for a transaction.
- **Counting transactions** — helps measure the size and activity of the UTXO set.
- **Available UTXOs** — includes outputs not already spent in the mempool.
---
## Summary
UTXOs provide a reliable and efficient way to manage cryptocurrency transactions:
- **Security** — prevents double-spending by tracking unspent outputs.
- **Transparency** — every UTXO can be independently verified.
- **Efficiency** — simplifies balance computation without maintaining account states.
- **Flexibility** — supports complex transaction types and multi-input outputs.
The UTXO model is a fundamental part of Bitcoin and other cryptocurrencies, ensuring secure, 
verifiable, and efficient handling of digital assets.