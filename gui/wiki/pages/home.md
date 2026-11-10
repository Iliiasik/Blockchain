# Introduction

This section is intended for the theoretical part of the application and for explaining the principles of blockchain operation. Every chapter describes one building block, shows the real code from this project and explains why it is written that way.

## How to read the wiki

The chapters follow the same order in which the data flows through the system:

1. **Blocks** and **Proof-of-Work** describe the container for data and the work needed to create it.
2. **Blockchain and storage** and **Binary encoding** show how blocks are linked and persisted with BoltDB.
3. **Transactions**, **Wallets** and **Base58Check** explain who owns coins and how ownership is proven.
4. **UTXO set**, **Merkle tree** and **Mempool** cover the optimizations a real node needs.
5. **P2P network** shows how several nodes agree on one chain.

Every code fragment is taken from the `core` package. Open the matching file on GitHub next to the article to see the full context.

## Try it while reading

The fastest way to learn is to follow the objectives on the Overview screen: create a wallet, mine the genesis block, mine a second block, send coins and confirm the transaction. Then come back and read about what just happened.

## Chapters
