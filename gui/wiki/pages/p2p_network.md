# Peer-to-Peer network in blockchain
A **Peer-to-Peer (P2P) network** is a decentralized system where every node (peer) performs 
identical functions, acting as both client and server.  
In blockchain, P2P networks maintain a distributed ledger — each node stores a complete 
blockchain copy and verifies data integrity through consensus with other nodes.
By eliminating central servers, P2P architecture provides:
- **Redundancy** — multiple data copies prevent single-point failures
- **Resilience** — network survives even if multiple nodes go offline
- **Direct communication** — nodes interact without intermediaries
---
{{image:p2p_network.png}}
---
## Usage in blockchain
P2P networks are foundational to Bitcoin and similar blockchains. Key characteristics:
- **Decentralization**  
  No central authority; validation occurs through collective node agreement.
- **Equal participation**  
  All nodes can propagate transactions, validate blocks, and maintain the ledger.
- **Data replication**  
  Full blockchain copies on each node enhance tamper resistance.
- **Consensus verification**  
  Transactions require multiple confirmations before permanent inclusion.
### Network types:
1. **Structured**
    - Nodes manage specific data segments
    - Efficient content lookup (e.g., DHTs)
2. **Unstructured**
    - Random peer connections
    - Easy node addition but slower searches
3. **Hybrid**
    - Balances centralized efficiency with decentralized robustness
Bitcoin's implementation:
- Transactions broadcast to all peers
- Mining nodes compete to add blocks
- No central entity controls ledger updates
---
## Summary
P2P networks enable blockchain's core advantages:
- **Decentralization**  
  Trustless operation without single points of control.
- **Security**  
  Distributed validation and cryptographic proofs prevent manipulation.
- **Scalability**  
  Dynamic node participation accommodates growth.
This architecture makes P2P networks indispensable for cryptocurrencies and Web3 systems.  