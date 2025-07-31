package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = "September 30th, 1998. It's a day I'll never forget. The cop inside me died that day."

type Blockchain struct {
	tip []byte
	Db  *bolt.DB
}

func CreateBlockchain(address string, subsidy int, targetBits int) (*Blockchain, error) {
	if dbExists() {
		return nil, fmt.Errorf("blockchain already exists")
	}

	var tip []byte

	cbtx := NewCoinbaseTX(address, genesisCoinbaseData, subsidy)

	genesis := NewGenesisBlock(cbtx, targetBits)

	db, err := bolt.Open(dbFile, 0600, nil)
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
		return nil, err
	}

	fmt.Printf("Blockchain created successfully!\n")
	fmt.Printf("Genesis block hash: %x\n", tip)
	fmt.Printf("Mining difficulty: %d bits\n", targetBits)
	fmt.Printf("Block reward (subsidy): %d\n", subsidy)

	bc := Blockchain{tip, db}
	return &bc, nil
}

func NewBlockchain() (*Blockchain, error) {
	if !dbExists() {
		return nil, fmt.Errorf("No existing blockchain found. Create one first")
	}

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, err
	}

	var tip []byte
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return fmt.Errorf("Blocks bucket not found")
		}
		tip = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		return nil, err
	}

	bc := Blockchain{tip, db}
	return &bc, nil
}

func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found")
}

func (bc *Blockchain) FindUTXO() map[string]TXOutputs {
	UTXO := make(map[string]TXOutputs)
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				// Was the output spent?
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			if tx.IsCoinbase() == false {
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

func (bc *Blockchain) MineBlock(transactions []*Transaction, currentBits int) *Block {
	var lastHash []byte
	var lastBits int

	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Panic("ERROR: Invalid transaction")
		}
	}

	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))

		if lastBlockData := b.Get(lastHash); lastBlockData != nil {
			lastBlock := DeserializeBlock(lastBlockData)
			lastBits = lastBlock.Bits
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	targetBits := currentBits
	if targetBits == 0 {
		targetBits = lastBits
	}

	newBlock := NewBlock(transactions, lastHash, targetBits)

	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if err := b.Put(newBlock.Hash, newBlock.Serialize()); err != nil {
			return err
		}
		if err := b.Put([]byte("l"), newBlock.Hash); err != nil {
			return err
		}
		bc.tip = newBlock.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return newBlock
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}
