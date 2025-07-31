package core

import (
	"bytes"
	"encoding/binary"
	"time"
)

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	Bits          int
}

func NewBlock(transactions []*Transaction, prevBlockHash []byte, targetBits int) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Nonce:         0,
		Bits:          targetBits,
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func NewGenesisBlock(coinbase *Transaction, targetBits int) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{}, targetBits)
}

func (b *Block) HashTransactions() []byte {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}
	mTree := NewMerkleTree(transactions)

	return mTree.RootNode.Data
}

func (b *Block) Serialize() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.LittleEndian, b.Timestamp)
	binary.Write(&buf, binary.LittleEndian, int64(len(b.Transactions)))

	for _, tx := range b.Transactions {
		txData := tx.Serialize()
		binary.Write(&buf, binary.LittleEndian, int64(len(txData)))
		buf.Write(txData)
	}

	binary.Write(&buf, binary.LittleEndian, int64(len(b.PrevBlockHash)))
	buf.Write(b.PrevBlockHash)

	binary.Write(&buf, binary.LittleEndian, int64(len(b.Hash)))
	buf.Write(b.Hash)

	binary.Write(&buf, binary.LittleEndian, int64(b.Nonce))
	binary.Write(&buf, binary.LittleEndian, int64(b.Bits))

	return buf.Bytes()
}

func DeserializeBlock(data []byte) *Block {
	buf := bytes.NewReader(data)
	block := Block{}

	binary.Read(buf, binary.LittleEndian, &block.Timestamp)

	var txCount int64
	binary.Read(buf, binary.LittleEndian, &txCount)
	block.Transactions = make([]*Transaction, txCount)

	for i := int64(0); i < txCount; i++ {
		var txLen int64
		binary.Read(buf, binary.LittleEndian, &txLen)
		txData := make([]byte, txLen)
		buf.Read(txData)
		block.Transactions[i] = DeserializeTransaction(txData)
	}

	var prevHashLen int64
	binary.Read(buf, binary.LittleEndian, &prevHashLen)
	block.PrevBlockHash = make([]byte, prevHashLen)
	buf.Read(block.PrevBlockHash)

	var hashLen int64
	binary.Read(buf, binary.LittleEndian, &hashLen)
	block.Hash = make([]byte, hashLen)
	buf.Read(block.Hash)

	var nonce int64
	binary.Read(buf, binary.LittleEndian, &nonce)
	block.Nonce = int(nonce)

	var bits int64
	binary.Read(buf, binary.LittleEndian, &bits)
	block.Bits = int(bits)

	return &block
}
