# Blockchain Demonstration

[![Stars](https://img.shields.io/github/stars/Iliiasik/Blockchain.svg?style=flat&logo=github)](https://github.com/Iliiasik/Blockchain/stargazers)

This application is created to demonstrate the principles of blockchain technology through a graphical user interface (GUI). It is built with Go using the [Fyne](https://fyne.io) toolkit.

You can either download the release, clone the project, or install it directly via fyne:

```bash
# clone
git clone https://github.com/Iliiasik/Blockchain.git

# run
go run main.go
```
Alternatively, you can install and run the application directly using fyne:

```bash
# Install the fyne CLI tool (if you don't have it yet)
go install fyne.io/tools/cmd/fyne@latest

# Install the Blockchain app via fyne
fyne install github.com/Iliiasik/Blockchain@latest
```

# Requirements

- Go 1.18 or higher
- GCC compiler (for OpenGL support used by Fyne)

# About

>A graphical blockchain simulator built with Go and Fyne.
>Visualizes core concepts like Proof-of-Work, UTXO-based transactions, mining, mempool, wallets, and block structure in an interactive and educational way.
> **[More →](https://github.com/Iliiasik/Blockchain/wiki)**

> P2P CLI version updated for the latest Go version is available in the [`p2p-cli`](https://github.com/Iliiasik/Blockchain/tree/p2p-cli) branch.

# Interface
## Main blockchain
![readme-part-1](https://github.com/user-attachments/assets/87c12218-0aa0-4f9f-8895-f86cd808865d)
---
![readme-part-2](https://github.com/user-attachments/assets/9dafd324-3835-43a4-a7cd-a216e1311323)
## P2P network simulation
![readme-part-3](https://github.com/user-attachments/assets/966e18a2-e259-47ab-a187-268ab39a314e)

