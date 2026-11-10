package core

import (
	"encoding/hex"
	"errors"

	bolt "go.etcd.io/bbolt"
	berrors "go.etcd.io/bbolt/errors"
)

const utxoBucket = "chainstate"

type UTXOSet struct {
	Blockchain *Blockchain
}

type UnspentOutput struct {
	TxID  []byte
	Index int
	TXOutput
	Pending bool
}

func (u UTXOSet) FindSpendableOutputs(pubkeyHash []byte, amount int) (int, map[string][]int, error) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0

	err := u.forEachOutput(func(txID []byte, outIdx int, out TXOutput) bool {
		if accumulated >= amount {
			return false
		}
		id := hex.EncodeToString(txID)
		if out.IsLockedWithKey(pubkeyHash) && !u.Blockchain.Mempool.IsOutputSpent(id, outIdx) {
			accumulated += out.Value
			unspentOutputs[id] = append(unspentOutputs[id], outIdx)
		}
		return true
	})

	return accumulated, unspentOutputs, err
}

func (u UTXOSet) FindUTXO(pubKeyHash []byte) ([]UnspentOutput, error) {
	var UTXOs []UnspentOutput

	err := u.forEachOutput(func(txID []byte, outIdx int, out TXOutput) bool {
		if pubKeyHash == nil || out.IsLockedWithKey(pubKeyHash) {
			UTXOs = append(UTXOs, UnspentOutput{
				TxID:     append([]byte{}, txID...),
				Index:    outIdx,
				TXOutput: out,
				Pending:  u.Blockchain.Mempool.IsOutputSpent(hex.EncodeToString(txID), outIdx),
			})
		}
		return true
	})

	return UTXOs, err
}

func (u UTXOSet) Balances() (map[string]int, error) {
	balances := make(map[string]int)

	err := u.forEachOutput(func(_ []byte, _ int, out TXOutput) bool {
		balances[string(out.PubKeyHash)] += out.Value
		return true
	})

	return balances, err
}

func (u UTXOSet) CountTransactions() (int, error) {
	counter := 0

	err := u.Blockchain.Db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(utxoBucket)); b != nil {
			counter = b.Stats().KeyN
		}
		return nil
	})

	return counter, err
}

func (u UTXOSet) Reindex() error {
	db := u.Blockchain.Db
	bucketName := []byte(utxoBucket)
	UTXO := u.Blockchain.FindUTXO()

	return db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(bucketName); err != nil && !errors.Is(err, berrors.ErrBucketNotFound) {
			return err
		}

		b, err := tx.CreateBucket(bucketName)
		if err != nil {
			return err
		}

		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			if err != nil {
				return err
			}

			if err := b.Put(key, outs.Serialize()); err != nil {
				return err
			}
		}

		return nil
	})
}

func (u UTXOSet) Update(block *Block) error {
	return u.Blockchain.Db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(utxoBucket))
		if err != nil {
			return err
		}

		for _, tx := range block.Transactions {
			if !tx.IsCoinbase() {
				for _, vin := range tx.Vin {
					outs := DeserializeOutputs(b.Get(vin.Txid))
					unspent := 0

					for outIdx := range outs.Outputs {
						if outIdx == vin.Vout {
							outs.Outputs[outIdx] = TXOutput{}
						}
						if len(outs.Outputs[outIdx].PubKeyHash) > 0 {
							unspent++
						}
					}

					if unspent == 0 {
						err = b.Delete(vin.Txid)
					} else {
						err = b.Put(vin.Txid, outs.Serialize())
					}
					if err != nil {
						return err
					}
				}
			}

			newOutputs := TXOutputs{Outputs: append([]TXOutput{}, tx.Vout...)}
			if err := b.Put(tx.ID, newOutputs.Serialize()); err != nil {
				return err
			}
		}

		return nil
	})
}

func (u UTXOSet) PruneMempool() (int, error) {
	var stale []string

	for _, tx := range u.Blockchain.Mempool.GetTransactions() {
		unspent, err := u.inputsUnspent(tx)
		if err != nil {
			return 0, err
		}
		if !unspent || !u.Blockchain.VerifyTransaction(tx) {
			stale = append(stale, hex.EncodeToString(tx.ID))
		}
	}

	return len(stale), u.Blockchain.Mempool.RemoveTransactions(stale...)
}

func (u UTXOSet) inputsUnspent(transaction *Transaction) (bool, error) {
	ok := true

	err := u.Blockchain.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		if b == nil {
			ok = false
			return nil
		}

		for _, vin := range transaction.Vin {
			outs := DeserializeOutputs(b.Get(vin.Txid))
			if vin.Vout < 0 || vin.Vout >= len(outs.Outputs) || len(outs.Outputs[vin.Vout].PubKeyHash) == 0 {
				ok = false
				return nil
			}
		}

		return nil
	})

	return ok, err
}

func (u UTXOSet) forEachOutput(fn func(txID []byte, outIdx int, out TXOutput) bool) error {
	return u.Blockchain.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			for outIdx, out := range DeserializeOutputs(v).Outputs {
				if len(out.PubKeyHash) == 0 {
					continue
				}
				if !fn(k, outIdx, out) {
					return nil
				}
			}
		}

		return nil
	})
}
