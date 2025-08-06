package core

import (
	"encoding/hex"
	"os"
	"strings"
	"sync"
)

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
		filePath:     "mempool.dat",
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

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		data, err := hex.DecodeString(line)
		if err != nil {
			continue
		}

		tx := DeserializeTransaction(data)
		if tx != nil {
			mp.addTransactionInternal(tx)
		}
	}
}

func (mp *Mempool) saveToFile() error {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var builder strings.Builder
	for _, tx := range mp.transactions {
		builder.WriteString(hex.EncodeToString(tx.Serialize()))
		builder.WriteString("\n")
	}

	return os.WriteFile(mp.filePath, []byte(builder.String()), 0600)
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

func (mp *Mempool) AddTransaction(tx *Transaction) bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if len(mp.transactions) >= mp.maxSize {
		return false
	}

	txID := hex.EncodeToString(tx.ID)
	for _, input := range tx.Vin {
		inputTxID := hex.EncodeToString(input.Txid)
		if spendingTxID, exists := mp.spentOutputs[inputTxID][input.Vout]; exists {
			if spendingTxID != txID {
				return false
			}
		}
	}

	mp.addTransactionInternal(tx)
	go mp.saveToFile()
	return true
}

func (mp *Mempool) GetTransaction(txID string) (*Transaction, bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	tx, exists := mp.transactions[txID]
	return tx, exists
}

func (mp *Mempool) RemoveTransaction(txID string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	tx, exists := mp.transactions[txID]
	if !exists {
		return
	}

	delete(mp.transactions, txID)

	for _, input := range tx.Vin {
		inputTxID := hex.EncodeToString(input.Txid)
		if outputs, exists := mp.spentOutputs[inputTxID]; exists {
			delete(outputs, input.Vout)
			if len(outputs) == 0 {
				delete(mp.spentOutputs, inputTxID)
			}
		}
	}

	go mp.saveToFile()
}

func (mp *Mempool) GetTransactions() []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txs := make([]*Transaction, 0, len(mp.transactions))
	for _, tx := range mp.transactions {
		txs = append(txs, tx)
	}
	return txs
}

func (mp *Mempool) Clear() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.transactions = make(map[string]*Transaction)
	mp.spentOutputs = make(map[string]map[int]string)
	_ = os.Remove(mp.filePath)
}

func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return len(mp.transactions)
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

	if outputs, exists := mp.spentOutputs[txID]; exists {
		_, spent := outputs[vout]
		return spent
	}
	return false
}
