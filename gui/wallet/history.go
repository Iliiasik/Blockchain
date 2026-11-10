package wallet

import (
	"Blockchain/core"
	"bytes"
	"encoding/hex"
)

type entry struct {
	Kind         string
	Amount       int
	Fee          int
	Counterparty string
	TxID         []byte
	Height       int
	Time         int64
	Pending      bool
}

func history(blocks []*core.Block, mempool []*core.Transaction, pubKeyHash []byte) []entry {
	index := make(map[string]*core.Transaction)
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			index[hex.EncodeToString(tx.ID)] = tx
		}
	}

	var entries []entry
	for _, tx := range mempool {
		if e, ok := describe(tx, index, pubKeyHash); ok {
			e.Pending = true
			entries = append(entries, e)
		}
	}

	for i, block := range blocks {
		for _, tx := range block.Transactions {
			if e, ok := describe(tx, index, pubKeyHash); ok {
				e.Height = len(blocks) - 1 - i
				e.Time = block.Timestamp
				entries = append(entries, e)
			}
		}
	}

	return entries
}

func describe(tx *core.Transaction, index map[string]*core.Transaction, pubKeyHash []byte) (entry, bool) {
	spent, received := 0, 0
	counterparty := ""

	if !tx.IsCoinbase() {
		for _, in := range tx.Vin {
			prev, ok := index[hex.EncodeToString(in.Txid)]
			if !ok || in.Vout < 0 || in.Vout >= len(prev.Vout) {
				continue
			}
			if prev.Vout[in.Vout].IsLockedWithKey(pubKeyHash) {
				spent += prev.Vout[in.Vout].Value
			}
		}
	}

	for _, out := range tx.Vout {
		if out.IsLockedWithKey(pubKeyHash) {
			received += out.Value
		} else if counterparty == "" {
			counterparty = core.PubKeyHashToAddress(out.PubKeyHash)
		}
	}

	e := entry{TxID: tx.ID, Fee: tx.Fee}
	switch {
	case spent > 0:
		e.Kind = "sent"
		e.Amount = received - spent
		e.Counterparty = counterparty
	case received > 0 && tx.IsCoinbase():
		e.Kind = "mined"
		e.Amount = received
		e.Counterparty = "block reward"
	case received > 0:
		e.Kind = "received"
		e.Amount = received
		e.Counterparty = tx.GetSenderAddress()
		if bytes.Equal(core.AddressToPubKeyHash(e.Counterparty), pubKeyHash) {
			e.Counterparty = "self"
		}
	default:
		return e, false
	}

	return e, true
}
