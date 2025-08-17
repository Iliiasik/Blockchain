# Blockchain structure and function
A blockchain is a distributed, ordered, and verifiable database composed of blocks, 
each containing valuable data and metadata. It ensures security, immutability, 
and decentralized trust in systems like Bitcoin.
---
{{image:blockchain.png}}
---
## Key components of a blockchain
- **Blocks** — the fundamental units storing transactions, timestamps, and cryptographic hashes.
- **Genesis Block** — the first block in the chain, created without a previous hash. 
All other blocks reference it.
- **PrevBlockHash** — links each block to its predecessor, forming a secure chain.
- **Hash** — a SHA-256 hash of the block ensuring immutability and integrity.
- **Proof-of-Work Fields (Nonce / Bits)** — ensure computational difficulty for adding new blocks.
- **UTXO Set** — unspent transaction outputs, used to track spendable balances.
- **Transactions** — data contained in blocks; can be standard or coinbase (mining reward) transactions.
---
## Usage in blockchain
Blockchains maintain a secure and verifiable ledger with the following mechanisms:
### Adding new blocks
- Each new block references the previous block's hash.
- Transactions must be validated before inclusion.
- Mining a block involves Proof-of-Work, a computationally intensive process to prevent tampering.
- After successful mining, the block is persisted in the blockchain database and the UTXO set is updated.
### Transaction verification and signing
- Transactions reference previous transactions via inputs.
- Non-coinbase transactions are signed with the sender's private key and verified 
against previous transaction outputs.
### Consensus and decentralization
- Distributed nodes validate and approve blocks.
- Decentralized validation ensures trust without a single decision maker.
### Database storage
- Blocks are stored in a persistent database (e.g., BoltDB).
- The latest block hash is maintained for easy chain extension and iteration.
- A mempool temporarily holds unconfirmed transactions before mining.
---
## Summary
Blockchains provide:
- **Security** — cryptographic hashes and Proof-of-Work prevent tampering.
- **Integrity & Traceability** — each block links to its predecessor, forming an ordered chain.
- **Decentralized Validation** — multiple nodes confirm transactions and blocks.
- **Efficient UTXO Management** — enables verification of spendable outputs and balances.
Blockchains are the backbone of decentralized cryptocurrencies, providing a secure, ordered, 
and verifiable system for storing transactions and other critical data.