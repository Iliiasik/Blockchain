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
	return PubKeyHashToAddress(HashPubKey(pubKey))
}

func PubKeyHashToAddress(pubKeyHash []byte) string {
	versionedPayload := append([]byte{version}, pubKeyHash...)
	fullPayload := append(versionedPayload, checksum(versionedPayload)...)

	return string(Base58Encode(fullPayload))
}

func AddressToPubKeyHash(address string) []byte {
	if !ValidateAddress(address) {
		return nil
	}

	pubKeyHash := Base58Decode([]byte(address))

	return pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
}
