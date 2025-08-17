# Mempool in blockchain
A **Mempool** (Memory Pool) is a temporary storage area for unconfirmed transactions in a blockchain
node. It holds transactions that have been received by the node but not yet included in a block.  
Mempools ensure that transactions can be propagated across the network efficiently and  
prevent double-spending before they are mined.  
Transactions in the mempool are tracked using their **transaction IDs**, and outputs that  
have already been spent are monitored to maintain consistency and validity.
---
{{image:mempool.png}}
---
## Usage in blockchain
Mempools play a critical role in the operation of blockchain nodes:
- **Transaction buffering** — holds unconfirmed transactions until they can be mined into a block.
- **Double-spend prevention** — tracks spent outputs to reject conflicting transactions.
- **Network propagation** — allows nodes to relay transactions to peers before they are included in blocks.
- **Resource management** — limits the number of transactions to avoid overloading the node (`maxSize`).
- **Persistence** — transactions can be serialized and saved to disk for recovery on restart.  
  This mechanism enables nodes to efficiently select valid transactions for mining while  
  ensuring the integrity of the blockchain state.
---
## Summary
Mempools are essential for the smooth operation of blockchain networks, combining:
- **Efficiency** — enables quick access to unconfirmed transactions for mining and validation.
- **Security** — prevents double-spending by tracking outputs already used in other transactions.
- **Network consistency** — allows rapid transaction propagation among nodes.  
  These principles make the mempool a core component of transaction handling in Bitcoin  
  and other decentralized systems.