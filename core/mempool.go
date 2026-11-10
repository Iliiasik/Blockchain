package core

import (
	"encoding/hex"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
)

const mempoolFile = "mempool.dat"

var ErrOutputReserved = errors.New("output is already spent by a pending transaction")

type Mempool struct {
	transactions map[string]*Transaction
	maxSize      int
	filePath     string
	mu           sync.RWMutex
	spentOutputs map[string]map[int]string
}

func NewMempool(maxSize int) *Mempool {
	mp := &Mempool{
		transactions: make(map[string]*Transaction),
		maxSize:      maxSize,
		filePath:     mempoolFile,
		spentOutputs: make(map[string]map[int]string),
	}

	mp.loadFromFile()

	return mp
}

func (mp *Mempool) loadFromFile() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	data, err := os.ReadFile(mp.filePath)
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}

		raw, err := hex.DecodeString(line)
		if err != nil {
			continue
		}

		mp.addTransactionInternal(DeserializeTransaction(raw))
	}
}

func (mp *Mempool) saveLocked() error {
	if len(mp.transactions) == 0 {
		err := os.Remove(mp.filePath)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := make([]string, 0, len(mp.transactions))
	for _, tx := range mp.transactions {
		lines = append(lines, hex.EncodeToString(tx.Serialize()))
	}
	sort.Strings(lines)

	return os.WriteFile(mp.filePath, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

func (mp *Mempool) addTransactionInternal(tx *Transaction) {
	txID := hex.EncodeToString(tx.ID)
	mp.transactions[txID] = tx

	for _, input := range tx.Vin {
		inputTxID := hex.EncodeToString(input.Txid)
		if mp.spentOutputs[inputTxID] == nil {
			mp.spentOutputs[inputTxID] = make(map[int]string)
		}
		mp.spentOutputs[inputTxID][input.Vout] = txID
	}
}

func (mp *Mempool) AddTransaction(tx *Transaction) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if len(mp.transactions) >= mp.maxSize {
		return ErrMempoolFull
	}

	txID := hex.EncodeToString(tx.ID)
	for _, input := range tx.Vin {
		inputTxID := hex.EncodeToString(input.Txid)
		if spendingTxID, exists := mp.spentOutputs[inputTxID][input.Vout]; exists && spendingTxID != txID {
			return ErrOutputReserved
		}
	}

	mp.addTransactionInternal(tx)

	return mp.saveLocked()
}

func (mp *Mempool) GetTransaction(txID string) (*Transaction, bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	tx, exists := mp.transactions[txID]
	return tx, exists
}

func (mp *Mempool) RemoveTransactions(txIDs ...string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	removed := false
	for _, txID := range txIDs {
		tx, exists := mp.transactions[txID]
		if !exists {
			continue
		}

		delete(mp.transactions, txID)
		removed = true

		for _, input := range tx.Vin {
			inputTxID := hex.EncodeToString(input.Txid)
			if outputs, exists := mp.spentOutputs[inputTxID]; exists {
				delete(outputs, input.Vout)
				if len(outputs) == 0 {
					delete(mp.spentOutputs, inputTxID)
				}
			}
		}
	}

	if !removed {
		return nil
	}

	return mp.saveLocked()
}

func (mp *Mempool) GetTransactions() []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txs := make([]*Transaction, 0, len(mp.transactions))
	for _, tx := range mp.transactions {
		txs = append(txs, tx)
	}

	sort.Slice(txs, func(i, j int) bool {
		if txs[i].Fee != txs[j].Fee {
			return txs[i].Fee > txs[j].Fee
		}
		return hex.EncodeToString(txs[i].ID) < hex.EncodeToString(txs[j].ID)
	})

	return txs
}

func (mp *Mempool) Clear() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.transactions = make(map[string]*Transaction)
	mp.spentOutputs = make(map[string]map[int]string)

	return mp.saveLocked()
}

func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return len(mp.transactions)
}

func (mp *Mempool) MaxSize() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return mp.maxSize
}

func (mp *Mempool) IsFull() bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return len(mp.transactions) >= mp.maxSize
}

func (mp *Mempool) SetMaxSize(size int) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.maxSize = size
}

func (mp *Mempool) IsOutputSpent(txID string, vout int) bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	_, spent := mp.spentOutputs[txID][vout]
	return spent
}
