package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
)

const coordinateSize = 32

var (
	ErrUnknownWallet      = errors.New("sender wallet is not found in wallet.dat")
	ErrPendingTransaction = errors.New("wallet already has pending transaction")
	ErrMempoolFull        = errors.New("mempool is full")
	ErrNotEnoughFunds     = errors.New("not enough funds")
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
	Fee  int
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

func (tx *Transaction) Serialize() []byte {
	buf := writeBytes(nil, tx.ID)

	buf = writeInt(buf, int64(len(tx.Vin)))
	for _, input := range tx.Vin {
		buf = writeBytes(buf, input.Txid)
		buf = writeInt(buf, int64(input.Vout))
		buf = writeBytes(buf, input.Signature)
		buf = writeBytes(buf, input.PubKey)
	}

	buf = writeInt(buf, int64(len(tx.Vout)))
	for _, output := range tx.Vout {
		buf = writeInt(buf, int64(output.Value))
		buf = writeBytes(buf, output.PubKeyHash)
	}

	return writeInt(buf, int64(tx.Fee))
}

func DeserializeTransaction(data []byte) *Transaction {
	buf := bytes.NewReader(data)
	tx := Transaction{}

	tx.ID = readBytes(buf)

	vinCount := readInt(buf)
	for i := int64(0); i < vinCount && buf.Len() > 0; i++ {
		var input TXInput
		input.Txid = readBytes(buf)
		input.Vout = int(readInt(buf))
		input.Signature = readBytes(buf)
		input.PubKey = readBytes(buf)
		tx.Vin = append(tx.Vin, input)
	}

	voutCount := readInt(buf)
	for i := int64(0); i < voutCount && buf.Len() > 0; i++ {
		var output TXOutput
		output.Value = int(readInt(buf))
		output.PubKeyHash = readBytes(buf)
		tx.Vout = append(tx.Vout, output)
	}

	tx.Fee = int(readInt(buf))

	return &tx
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) error {
	if tx.IsCoinbase() {
		return nil
	}

	if err := checkPrevTransactions(tx, prevTXs); err != nil {
		return err
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		if err != nil {
			return err
		}

		signature := make([]byte, 2*coordinateSize)
		r.FillBytes(signature[:coordinateSize])
		s.FillBytes(signature[coordinateSize:])

		tx.Vin[inID].Signature = signature
	}

	return nil
}

func checkPrevTransactions(tx *Transaction, prevTXs map[string]Transaction) error {
	for _, vin := range tx.Vin {
		prevTx, ok := prevTXs[hex.EncodeToString(vin.Txid)]
		if !ok || prevTx.ID == nil {
			return fmt.Errorf("previous transaction %x is not found", vin.Txid)
		}
		if vin.Vout < 0 || vin.Vout >= len(prevTx.Vout) {
			return fmt.Errorf("previous transaction %x has no output %d", vin.Txid, vin.Vout)
		}
	}

	return nil
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))

	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	lines = append(lines, fmt.Sprintf("     Fee: %d", tx.Fee))

	return strings.Join(lines, "\n")
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs, tx.Fee}
	return txCopy
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	if checkPrevTransactions(tx, prevTXs) != nil {
		return false
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		if len(vin.Signature) == 0 || !vin.UsesKey(prevTx.Vout[vin.Vout].PubKeyHash) {
			return false
		}

		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		rawPubKey, err := ecdsa.ParseUncompressedPublicKey(curve, append([]byte{4}, vin.PubKey...))
		if err != nil {
			return false
		}
		if !ecdsa.Verify(rawPubKey, txCopy.ID, &r, &s) {
			return false
		}
	}

	return true
}

func NewCoinbaseTX(to, data string, subsidy int) *Transaction {
	if data == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}
		data = fmt.Sprintf("%x", randData)
	}

	txin := TXInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}, 0}
	tx.ID = tx.Hash()

	return &tx
}

func NewUTXOTransaction(from, to string, amount int, fee int, UTXOSet *UTXOSet) (*Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput

	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if fee < 0 {
		return nil, fmt.Errorf("fee can not be negative")
	}
	if !ValidateAddress(from) || !ValidateAddress(to) {
		return nil, fmt.Errorf("invalid address")
	}

	wallets, err := NewWallets()
	if err != nil {
		return nil, err
	}
	wallet, ok := wallets.GetWallet(from)
	if !ok {
		return nil, ErrUnknownWallet
	}
	pubKeyHash := HashPubKey(wallet.PublicKey)

	mempool := UTXOSet.Blockchain.Mempool
	for _, tx := range mempool.GetTransactions() {
		if tx.GetSenderAddress() == from {
			return nil, ErrPendingTransaction
		}
	}

	if mempool.IsFull() {
		return nil, ErrMempoolFull
	}

	acc, validOutputs, err := UTXOSet.FindSpendableOutputs(pubKeyHash, amount+fee)
	if err != nil {
		return nil, err
	}

	if acc < amount+fee {
		return nil, fmt.Errorf("%w: need %d, available %d", ErrNotEnoughFunds, amount+fee, acc)
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			return nil, err
		}

		for _, out := range outs {
			input := TXInput{txID, out, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount+fee {
		outputs = append(outputs, *NewTXOutput(acc-amount-fee, from))
	}

	tx := Transaction{nil, inputs, outputs, fee}
	tx.ID = tx.Hash()
	if err := UTXOSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey); err != nil {
		return nil, err
	}

	if err := mempool.AddTransaction(&tx); err != nil {
		return nil, err
	}

	return &tx, nil
}
