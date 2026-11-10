package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"log"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)
const walletFile = "wallet.dat"
const addressChecksumLen = 4

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() *Wallet {
	private, public := newKeyPair()
	wallet := Wallet{private, public}

	return &wallet
}

func (w Wallet) GetAddress() []byte {
	return []byte(PubKeyToAddress(w.PublicKey))
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, _ = RIPEMD160Hasher.Write(publicSHA256[:])

	return RIPEMD160Hasher.Sum(nil)
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))

	if len(pubKeyHash) != 1+ripemd160.Size+addressChecksumLen || pubKeyHash[0] != version {
		return false
	}

	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	targetChecksum := checksum(pubKeyHash[:len(pubKeyHash)-addressChecksumLen])

	return bytes.Equal(actualChecksum, targetChecksum)
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	point, err := private.PublicKey.Bytes()
	if err != nil {
		log.Panic(err)
	}

	return *private, point[1:]
}

type _PrivateKey struct {
	D          *big.Int
	PublicKeyX *big.Int
	PublicKeyY *big.Int
}

func (w *Wallet) GobEncode() ([]byte, error) {
	secret, err := w.PrivateKey.Bytes()
	if err != nil {
		return nil, err
	}
	point, err := w.PrivateKey.PublicKey.Bytes()
	if err != nil {
		return nil, err
	}
	privKey := &_PrivateKey{
		D:          new(big.Int).SetBytes(secret),
		PublicKeyX: new(big.Int).SetBytes(point[1 : 1+coordinateSize]),
		PublicKeyY: new(big.Int).SetBytes(point[1+coordinateSize:]),
	}

	var buf bytes.Buffer

	encoder := gob.NewEncoder(&buf)
	err = encoder.Encode(privKey)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(w.PublicKey)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (w *Wallet) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	var privKey _PrivateKey

	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(&privKey)
	if err != nil {
		return err
	}

	if privKey.D == nil || privKey.D.Sign() <= 0 || privKey.D.BitLen() > 8*coordinateSize {
		return errors.New("wallet holds an invalid private key")
	}
	key, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), privKey.D.FillBytes(make([]byte, coordinateSize)))
	if err != nil {
		return err
	}
	w.PrivateKey = *key
	w.PublicKey = make([]byte, buf.Len())
	_, err = buf.Read(w.PublicKey)
	if err != nil {
		return err
	}

	return nil
}
