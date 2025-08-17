# Base58 in blockchain
Base58 is a binary-to-text encoding scheme designed to represent large numbers using a set 
of 58 alphanumeric characters. Unlike Base64, it deliberately omits characters that may cause
confusion when visually inspected or manually typed. Specifically, 
the following characters are excluded:
- `0` (zero)
- `O` (uppercase O)
- `l` (lowercase L)
- `I` (uppercase I)
- `+` (plus sign)
- `/` (forward slash)
By removing these ambiguous symbols, Base58 provides a more
human-friendly format that reduces the risk of transcription errors.
---
{{image:base58.png}}
---
## Usage in blockchain
In blockchain systems, Base58 is widely used to encode addresses and identifiers. 
It ensures compactness, readability, and reliable cross-platform compatibility.
A notable example is Bitcoin, where addresses are encoded using
**Base58Check** — a variant of Base58 that adds a checksum.
This checksum introduces an extra layer of error detection, which allows wallets 
and applications to catch common mistakes before processing any transaction.
---
## Summary
Base58 plays a crucial role in blockchain applications by balancing two main aspects 
of data handling and usability:
- **Machine efficiency** — compact encoding of binary data
- **Human usability** — avoiding confusing characters and reducing errors
This combination makes Base58 a practical and reliable encoding
scheme for cryptocurrencies and decentralized systems.