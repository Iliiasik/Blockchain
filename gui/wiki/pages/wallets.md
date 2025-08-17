# Blockchain wallets
A blockchain wallet is a digital wallet that allows users to store, send, 
and receive cryptocurrencies securely. It relies on public-key cryptography, 
generating a unique private key and public key for every wallet. The wallet 
does not store coins directly but manages the cryptographic keys that control access
to funds stored on the blockchain.
---
{{image:wallets.png}}
---
## How blockchain wallets work
In blockchain systems, wallets play a vital role in identity, transaction validation, 
and fund management:
- **Key Pair Generation** — wallets create an asymmetric key pair (private and public key) 
using elliptic curve cryptography (ECC)
- **Private Key** — acts as the secret credential that must be kept secure; 
it allows the signing of transactions
- **Public Key** — derived from the private key and used to generate addresses that others 
use to send funds
- **Addresses** — user-friendly representations (e.g., Base58Check encoding in Bitcoin) 
of the public key hash
## Wallet storage and management
Wallets can generate and manage multiple addresses:
- Multiple addresses can be stored in a single wallet
- New wallets can be created and added to the collection
- All addresses managed by the wallet can be listed
- Wallets support persistence to files for recovery while keeping private keys secure
## Security mechanisms
Blockchain wallets implement multiple security layers:
- **Checksums** — detect typing/transmission errors in addresses
- **Hashing (SHA-256 + RIPEMD-160)** — ensures integrity and reduces public key size
- **Elliptic Curve Cryptography (P256)** — provides strong cryptographic security
- **Serialization** — allows secure storage of key material
## Types of wallets
Blockchain wallets can be categorized as:
- **Hot wallets** — connected to the internet, convenient but more vulnerable
- **Cold wallets** — offline storage, highly secure for long-term holding
- **Hardware wallets** — physical devices securing private keys offline
- **Software wallets** — apps or programs running on user devices
In this application, the implemented wallet type is a **hot non-custodial software wallet**.
---
## Summary
Blockchain wallets provide essential cryptocurrency functionality:
- Establish digital ownership through private/public keys
- Enable transaction authorization
- Include error protection mechanisms
- Support long-term access to funds
- Allow management of multiple addresses
Blockchain wallets balance usability and security, enabling safe interaction with decentralized systems.