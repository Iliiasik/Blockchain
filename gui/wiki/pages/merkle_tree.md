# Merkle Tree in blockchain
A **Merkle Tree** is a binary tree of cryptographic hashes constructed from the
transaction data of a block. It provides a compact and secure way to verify
whether a specific transaction is included in a block — without needing to
download and process every transaction.
The structure is built recursively:
- Each **leaf node** contains the hash of a transaction.
- Each **parent node** is the hash of the concatenation of its two child nodes.
- The process continues until a single hash remains — the **Merkle Root**.
If a level of the tree contains an odd number of nodes, the last element is
duplicated to keep the tree balanced. This ensures that every non-leaf node has
exactly two children.
---
{{image:merkle_tree.png}}
---
## Usage in blockchain
Merkle Trees play a fundamental role in blockchain systems, particularly in Bitcoin:
- **Compactness** — A Merkle Root summarizes potentially thousands of transactions
  into a single 32-byte value stored in the block header.
- **Verification** — Through Merkle proofs, light (SPV) clients can confirm
  transaction inclusion without downloading the full block.
- **Security** — Any modification to a transaction alters its hash, which propagates
  up the tree, changing the Merkle Root and invalidating the block.
Bitcoin specifically uses **double SHA-256 hashing** for every node in the tree.
Hashes are often represented in little-endian format, which requires byte-order
reversal in some implementations.
---
## Example
Consider a block with three transactions **a, b, c**:  
d1 = dhash(a)  
d2 = dhash(b)  
d3 = dhash(c)  
d4 = dhash(c) # duplicate last, because odd number of transactions  
d5 = dhash(d1 || d2)  
d6 = dhash(d3 || d4)
Merkle Root = dhash(d5 || d6)
Here `dhash(x)` means **double SHA-256** of the input.
---
## Merkle proofs
A **Merkle Proof** allows verifying the inclusion of a specific transaction in a
block without downloading the entire block.
- The proof consists of the transaction hash and a minimal set of sibling hashes
  needed to reconstruct the Merkle Root.
- By successively hashing the transaction with the provided siblings, a lightweight
  client (SPV) can check whether the calculated root matches the block header’s
  Merkle Root.
This makes Merkle Trees essential for **Simplified Payment Verification (SPV)**
in Bitcoin and other blockchains.
---
## Summary
Merkle Trees are essential to blockchain design, combining three crucial aspects:
- **Efficiency** — enable lightweight verification without the full dataset.
- **Security** — prevent tampering by binding all transactions to a single root hash.
- **Scalability** — allow simplified clients (SPV) to operate securely with minimal
  resources.
This makes Merkle Trees a core component of Bitcoin and many other decentralized
systems.