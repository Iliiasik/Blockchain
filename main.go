package main

import (
	"Blockchain/controllers"
	"fmt"
)

func main() {
	bc := controllers.NewBlockchain()

	bc.AddBlock("Send 1 BTC to Iliias")
	bc.AddBlock("Send 2 more BTC to Iliias")

	for _, block := range bc.Blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Println()
	}
}
