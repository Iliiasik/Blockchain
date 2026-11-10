# Blockchain learning GUI

This application is created to demonstrate the principles of blockchain technology through a graphical user interface (GUI). It is built with Go using the [Fyne](https://fyne.io) toolkit.

A graphical blockchain simulator that visualizes core concepts like **Proof-of-Work**, **UTXO-based transactions**, **mining**, **mempool**, **wallets**, and **block structure** in an interactive and educational way.

<img width="100%" alt="Overview" src="https://github.com/user-attachments/assets/5cb77ce3-758b-4685-9fa5-73cd0f6fd700" />

## Getting started

You can either download the [release](https://github.com/Iliiasik/Blockchain/releases/latest), clone the project, or install it directly via fyne.

**Clone and run**

```bash
git clone https://github.com/Iliiasik/Blockchain.git
cd Blockchain
go run main.go
```

**Install with fyne**

```bash
go install fyne.io/tools/cmd/fyne@latest
fyne install github.com/Iliiasik/Blockchain@latest
```

### Requirements

- Go 1.26 or higher
- GCC compiler (for OpenGL support used by Fyne)

> [!TIP]
> The release builds need neither: download the archive for your system and run it.

## Interface

<img width="100%" alt="Create blockchain" src="https://github.com/user-attachments/assets/33000d38-969a-4dc0-af89-e9ff229ce7c1" />

---

<img width="100%" alt="Wallets" src="https://github.com/user-attachments/assets/29a051dc-4905-406a-8ca2-35759a5c7d43" />

---

<img width="100%" alt="Wallets 2" src="https://github.com/user-attachments/assets/f44fc860-2fff-4a68-866b-8d0b3fc93d29" />

---

<img width="100%" alt="Blockchain" src="https://github.com/user-attachments/assets/b69c5f2d-998f-46b6-a3a6-61861d490475" />

---

<img width="100%" alt="Blockchain Light" src="https://github.com/user-attachments/assets/ed6ff0bd-0f6b-4bdd-a5b4-2dca10629d55" />

---

<img width="100%" alt="Mining" src="https://github.com/user-attachments/assets/bdf66ee2-1bee-42d9-bf42-5a1d80f60380" />

---

<img width="100%" alt="Transactions" src="https://github.com/user-attachments/assets/e2d91b88-53c3-4d1f-bd7f-8d741f2c494a" />

---

<img width="100%" alt="Transactions 2" src="https://github.com/user-attachments/assets/3dc7d719-80be-479c-a5e4-0f60e24f41f5" />

---

<img width="100%" alt="UTXO" src="https://github.com/user-attachments/assets/16b57911-115d-4c6d-90f8-b837e5c33fea" />

---

<img width="100%" alt="Visualizer" src="https://github.com/user-attachments/assets/48d1e7b5-1be0-499d-870f-7aa119d603a0" />

---

<img width="100%" alt="Visualizer Light" src="https://github.com/user-attachments/assets/03c43b28-fb39-4215-aa28-0a0f9e5a57a1" />

---

<img width="100%" alt="Visualizer Guide" src="https://github.com/user-attachments/assets/b783f1a2-91bb-47c7-87fb-7cba282f560b" />

---

<img width="100%" alt="P2P network" src="https://github.com/user-attachments/assets/99dd6c5c-7149-4662-945b-7d5c14bec505" />

---

<img width="100%" alt="P2P network 2" src="https://github.com/user-attachments/assets/ea1dd7ce-d84e-4c2e-bf61-679693222913" />

---

<img width="100%" alt="Wiki" src="https://github.com/user-attachments/assets/aedcfd96-cf6f-4454-8f9e-1bf7a326ffbb" />

---

<img width="100%" alt="Wiki Blocks" src="https://github.com/user-attachments/assets/f4aec03b-4cf8-4acb-8eb0-5b60a7865e6d" />

---

<img width="100%" alt="Wiki Blocks 2" src="https://github.com/user-attachments/assets/14a275ea-0501-4840-973f-3924eb0c318d" />

---

<img width="100%" alt="Wiki Merkle tree" src="https://github.com/user-attachments/assets/56bac055-bfe1-419a-a9e0-ef9249eea895" />

## Credits

- The built-in wiki is based on the [Building Blockchain in Go](https://jeiwan.net/posts/building-blockchain-in-go-part-1/) tutorial series by Ivan Kuznetsov ([Jeiwan](https://github.com/Jeiwan/blockchain_go)).
- The diagrams come from the [Bitcoin Developer Guide](https://developer.bitcoin.org/devguide/) ([source](https://github.com/bitcoin-dot-org/developer.bitcoin.org), MIT License).
- The icons come from [SVG Repo](https://www.svgrepo.com/svg/380472/blockchain-digital-future-system-security) (CC0 License).
- The interface font is [Sora](https://fonts.google.com/specimen/Sora), licensed under the SIL Open Font License.

## License

This project is open source under the [MIT License](LICENSE) and has no commercial intent. It is made purely for learning.

Feel free to contribute, fork the project, or use it for your own purposes within the terms of the license.

---

<p align="center">⭐ If this project was helpful, leave a star</p>
