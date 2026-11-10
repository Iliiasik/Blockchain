package core

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
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

func NewBlock(transactions []*Transaction, prevBlockHash []byte, targetBits int) (*Block, error) {
	return NewBlockContext(context.Background(), transactions, prevBlockHash, targetBits, nil)
}

func NewBlockContext(ctx context.Context, transactions []*Transaction, prevBlockHash []byte, targetBits int, progress *MiningProgress) (*Block, error) {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Nonce:         0,
		Bits:          targetBits,
	}

	pow := NewProofOfWork(block)
	nonce, hash, err := pow.Run(ctx, progress)
	if err != nil {
		return nil, err
	}

	block.Hash = hash
	block.Nonce = nonce

	return block, nil
}

func NewGenesisBlock(coinbase *Transaction, targetBits int) (*Block, error) {
	return NewBlock([]*Transaction{coinbase}, []byte{}, targetBits)
}

func (b *Block) HashTransactions() []byte {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}
	mTree := NewMerkleTree(transactions)
	if mTree.RootNode == nil {
		return nil
	}

	return mTree.RootNode.Data
}

func (b *Block) Serialize() []byte {
	buf := writeInt(nil, b.Timestamp)
	buf = writeInt(buf, int64(len(b.Transactions)))

	for _, tx := range b.Transactions {
		buf = writeBytes(buf, tx.Serialize())
	}

	buf = writeBytes(buf, b.PrevBlockHash)
	buf = writeBytes(buf, b.Hash)

	buf = writeInt(buf, int64(b.Nonce))
	buf = writeInt(buf, int64(b.Bits))

	return buf
}

func DeserializeBlock(data []byte) *Block {
	buf := bytes.NewReader(data)
	block := Block{}

	block.Timestamp = readInt(buf)

	txCount := readInt(buf)
	for i := int64(0); i < txCount && buf.Len() > 0; i++ {
		block.Transactions = append(block.Transactions, DeserializeTransaction(readBytes(buf)))
	}

	block.PrevBlockHash = readBytes(buf)
	block.Hash = readBytes(buf)
	block.Nonce = int(readInt(buf))
	block.Bits = int(readInt(buf))

	return &block
}

func writeInt(buf []byte, value int64) []byte {
	return binary.LittleEndian.AppendUint64(buf, uint64(value))
}

func writeBytes(buf []byte, data []byte) []byte {
	return append(writeInt(buf, int64(len(data))), data...)
}

func readInt(buf *bytes.Reader) int64 {
	var value int64
	if binary.Read(buf, binary.LittleEndian, &value) != nil {
		return 0
	}

	return value
}

func readBytes(buf *bytes.Reader) []byte {
	size := readInt(buf)
	if size <= 0 || size > int64(buf.Len()) {
		return []byte{}
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(buf, data); err != nil {
		return []byte{}
	}

	return data
}
