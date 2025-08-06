package core

func (tx *Transaction) GetSenderAddress() string {
	if tx.IsCoinbase() || len(tx.Vin) == 0 {
		return ""
	}
	return PubKeyToAddress(tx.Vin[0].PubKey)
}

func (tx *Transaction) GetRecipientAddress() string {
	if len(tx.Vout) == 0 {
		return ""
	}
	return PubKeyHashToAddress(tx.Vout[0].PubKeyHash)
}

func (tx *Transaction) GetAmount() int {
	if len(tx.Vout) == 0 {
		return 0
	}
	return tx.Vout[0].Value
}

func PubKeyToAddress(pubKey []byte) string {
	pubKeyHash := HashPubKey(pubKey)
	return string(Base58Encode(append([]byte{0x00}, pubKeyHash...)))
}

func PubKeyHashToAddress(pubKeyHash []byte) string {
	return string(Base58Encode(append([]byte{0x00}, pubKeyHash...)))
}
