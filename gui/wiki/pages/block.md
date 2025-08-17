# Blocks in blockchain
A block is the fundamental unit of a blockchain, storing both valuable data and metadata. 
In cryptocurrencies like Bitcoin, blocks primarily contain transactions, timestamps, 
and cryptographic hashes linking them to previous blocks.
## Key components of a block:
- **Timestamp** — the creation time of the block
- **Transactions / Data** — the main content stored in the block
- **PrevBlockHash** — hash of the previous block, linking blocks together
- **Hash** — SHA-256 hash of the block, ensuring integrity and immutability
- **Nonce / Bits** — fields used in Proof-of-Work to secure block addition
Blocks are organized sequentially, forming a back-linked list, which allows easy 
retrieval of the latest block and verification of the chain.
---
{{image:block.png}}
---
## Usage in blockchain
Blocks are used to maintain a secure, ordered, and verifiable ledger. Important points:
### Genesis Block
The first block in a blockchain, created without a previous hash. All other blocks reference it.
### Hashing
Blocks' hashes are calculated from their contents and previous block hash, providing immutability.
### Adding New Blocks
Each new block references the previous block's hash. In real blockchains, adding a block requires 
Proof-of-Work, a computationally intensive process, to prevent tampering.
### Consensus
Distributed blockchains require nodes to validate and approve blocks before addition, 
ensuring decentralized trust.
### Simplified Prototypes
A blockchain can be implemented as an array of blocks with back-links for simplicity; 
real implementations are more complex and include consensus and transaction handling.
---
## Summary
Blocks provide the following advantages:
- **Security** — cryptographic hashing ensures integrity and prevents modification
- **Order & Traceability** — sequential linking allows verification of the chain's history
- **Decentralized validation** — blocks are confirmed by multiple nodes in the network
- **Foundation for Proof-of-Work** — computational difficulty protects against attacks
Blocks are the core building blocks of any blockchain, enabling secure, verifiable, and ordered 
storage of transactions and other data.