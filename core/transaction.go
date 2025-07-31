package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/big"

	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

func (tx *Transaction) Serialize() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.LittleEndian, int64(len(tx.ID)))
	buf.Write(tx.ID)

	binary.Write(&buf, binary.LittleEndian, int64(len(tx.Vin)))
	for _, input := range tx.Vin {
		binary.Write(&buf, binary.LittleEndian, int64(len(input.Txid)))
		buf.Write(input.Txid)

		binary.Write(&buf, binary.LittleEndian, int64(input.Vout))

		binary.Write(&buf, binary.LittleEndian, int64(len(input.Signature)))
		buf.Write(input.Signature)

		binary.Write(&buf, binary.LittleEndian, int64(len(input.PubKey)))
		buf.Write(input.PubKey)
	}

	binary.Write(&buf, binary.LittleEndian, int64(len(tx.Vout)))
	for _, output := range tx.Vout {
		binary.Write(&buf, binary.LittleEndian, int64(output.Value))

		binary.Write(&buf, binary.LittleEndian, int64(len(output.PubKeyHash)))
		buf.Write(output.PubKeyHash)
	}

	return buf.Bytes()
}

func DeserializeTransaction(data []byte) *Transaction {
	buf := bytes.NewReader(data)
	tx := Transaction{}

	var idLen int64
	binary.Read(buf, binary.LittleEndian, &idLen)
	tx.ID = make([]byte, idLen)
	buf.Read(tx.ID)

	var vinCount int64
	binary.Read(buf, binary.LittleEndian, &vinCount)
	tx.Vin = make([]TXInput, vinCount)

	for i := range tx.Vin {
		var txidLen int64
		binary.Read(buf, binary.LittleEndian, &txidLen)
		tx.Vin[i].Txid = make([]byte, txidLen)
		buf.Read(tx.Vin[i].Txid)

		var vout int64
		binary.Read(buf, binary.LittleEndian, &vout)
		tx.Vin[i].Vout = int(vout)

		var sigLen int64
		binary.Read(buf, binary.LittleEndian, &sigLen)
		tx.Vin[i].Signature = make([]byte, sigLen)
		buf.Read(tx.Vin[i].Signature)

		var pubKeyLen int64
		binary.Read(buf, binary.LittleEndian, &pubKeyLen)
		tx.Vin[i].PubKey = make([]byte, pubKeyLen)
		buf.Read(tx.Vin[i].PubKey)
	}

	var voutCount int64
	binary.Read(buf, binary.LittleEndian, &voutCount)
	tx.Vout = make([]TXOutput, voutCount)

	for i := range tx.Vout {
		var val int64
		binary.Read(buf, binary.LittleEndian, &val)
		tx.Vout[i].Value = int(val)

		var pubKeyHashLen int64
		binary.Read(buf, binary.LittleEndian, &pubKeyHashLen)
		tx.Vout[i].PubKeyHash = make([]byte, pubKeyHashLen)
		buf.Read(tx.Vout[i].PubKeyHash)
	}

	return &tx
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
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
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
	}
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

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
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
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}}
	tx.ID = tx.Hash()

	return &tx
}

func NewUTXOTransaction(from, to string, amount int, UTXOSet *UTXOSet) (*Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput

	wallets, err := NewWallets()
	if err != nil {
		return nil, err
	}
	wallet := wallets.GetWallet(from)
	pubKeyHash := HashPubKey(wallet.PublicKey)
	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		return nil, fmt.Errorf("not enough funds")
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
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXOSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)

	return &tx, nil
}
