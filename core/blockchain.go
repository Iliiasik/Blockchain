package core

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = "September 30th, 1998. It's a day I'll never forget. The cop inside me died that day."

var ErrNoBlockchain = errors.New("no existing blockchain found. Create one first")

var dbOptions = &bolt.Options{Timeout: time.Second}

type Blockchain struct {
	tip     []byte
	Db      *bolt.DB
	Mempool *Mempool
}

func CreateBlockchain(address string, subsidy int, targetBits int) (*Blockchain, error) {
	if dbExists() {
		return nil, fmt.Errorf("blockchain already exists")
	}
	if !ValidateAddress(address) {
		return nil, fmt.Errorf("invalid genesis address")
	}

	var tip []byte

	cbtx := NewCoinbaseTX(address, genesisCoinbaseData, subsidy)
	genesis, err := NewGenesisBlock(cbtx, targetBits)
	if err != nil {
		return nil, err
	}

	db, err := bolt.Open(dbFile, 0600, dbOptions)
	if err != nil {
		return nil, fmt.Errorf("could not open db: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			return fmt.Errorf("could not create bucket: %v", err)
		}

		if err := b.Put(genesis.Hash, genesis.Serialize()); err != nil {
			return fmt.Errorf("could not store genesis block: %v", err)
		}

		if err := b.Put([]byte("l"), genesis.Hash); err != nil {
			return fmt.Errorf("could not update last block hash: %v", err)
		}

		tip = genesis.Hash
		return nil
	})
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}

	bc := Blockchain{tip, db, NewMempool(10)}
	if err := bc.Mempool.Clear(); err != nil {
		return nil, errors.Join(err, db.Close())
	}

	return &bc, nil
}

func NewBlockchain() (*Blockchain, error) {
	if !dbExists() {
		return nil, ErrNoBlockchain
	}

	db, err := bolt.Open(dbFile, 0600, dbOptions)
	if err != nil {
		return nil, fmt.Errorf("could not open db: %v", err)
	}

	var tip []byte
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return fmt.Errorf("blocks bucket not found")
		}
		tip = append([]byte{}, b.Get([]byte("l"))...)
		return nil
	})
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}

	bc := Blockchain{tip, db, NewMempool(10)}
	return &bc, nil
}

func RemoveBlockchain() error {
	if err := os.Remove(dbFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(mempoolFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (bc *Blockchain) Close() error {
	return bc.Db.Close()
}

func (bc *Blockchain) Blocks() []*Block {
	var blocks []*Block

	bci := bc.Iterator()
	for {
		block := bci.Next()
		if block == nil {
			break
		}
		blocks = append(blocks, block)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return blocks
}

func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()
		if block == nil {
			break
		}

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("transaction is not found")
}

func (bc *Blockchain) FindUTXO() map[string]TXOutputs {
	UTXO := make(map[string]TXOutputs)
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block := bci.Next()
		if block == nil {
			break
		}

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			outs := TXOutputs{Outputs: make([]TXOutput, len(tx.Vout))}
			unspent := 0
			for outIdx, out := range tx.Vout {
				if !containsIndex(spentTXOs[txID], outIdx) {
					outs.Outputs[outIdx] = out
					unspent++
				}
			}
			if unspent > 0 {
				UTXO[txID] = outs
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.Db}

	return bci
}

func (bc *Blockchain) MineBlock(ctx context.Context, transactions []*Transaction, targetBits int, progress *MiningProgress) (*Block, error) {
	if len(transactions) == 0 || !transactions[0].IsCoinbase() {
		return nil, errors.New("block must start with coinbase transaction")
	}

	for _, tx := range transactions[1:] {
		if !bc.VerifyTransaction(tx) {
			return nil, fmt.Errorf("invalid transaction: %x", tx.ID)
		}
	}

	var lastHash []byte
	var lastBits int

	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return errors.New("blocks bucket not found")
		}
		lastHash = append([]byte{}, b.Get([]byte("l"))...)

		if lastBlockData := b.Get(lastHash); lastBlockData != nil {
			lastBits = DeserializeBlock(lastBlockData).Bits
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error getting last block: %v", err)
	}

	if targetBits == 0 {
		targetBits = lastBits
	}

	newBlock, err := NewBlockContext(ctx, transactions, lastHash, targetBits, progress)
	if err != nil {
		return nil, err
	}

	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return errors.New("blocks bucket not found")
		}
		if !bytes.Equal(b.Get([]byte("l")), lastHash) {
			return errors.New("chain tip changed while mining")
		}
		if err := b.Put(newBlock.Hash, newBlock.Serialize()); err != nil {
			return err
		}
		return b.Put([]byte("l"), newBlock.Hash)
	})
	if err != nil {
		return nil, fmt.Errorf("error saving block: %v", err)
	}
	bc.tip = newBlock.Hash

	mined := make([]string, 0, len(transactions)-1)
	for _, tx := range transactions[1:] {
		mined = append(mined, hex.EncodeToString(tx.ID))
	}
	if err := bc.Mempool.RemoveTransactions(mined...); err != nil {
		return newBlock, fmt.Errorf("block saved, but the mempool file was not updated: %w", err)
	}

	return newBlock, nil
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) error {
	prevTXs, err := bc.previousTransactions(tx)
	if err != nil {
		return err
	}

	return tx.Sign(privKey, prevTXs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs, err := bc.previousTransactions(tx)
	if err != nil {
		return false
	}

	return tx.Verify(prevTXs)
}

func (bc *Blockchain) previousTransactions(tx *Transaction) (map[string]Transaction, error) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			return nil, err
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return prevTXs, nil
}

func containsIndex(indexes []int, index int) bool {
	for _, i := range indexes {
		if i == index {
			return true
		}
	}

	return false
}

func dbExists() bool {
	_, err := os.Stat(dbFile)

	return err == nil
}

func BlockchainExists() bool {
	return dbExists()
}
