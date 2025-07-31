package core

import (
	"bytes"
	"encoding/binary"
	"log"
)

type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}

type TXOutputs struct {
	Outputs []TXOutput
}

func (outs TXOutputs) Serialize() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.LittleEndian, int64(len(outs.Outputs)))

	for _, out := range outs.Outputs {
		binary.Write(&buf, binary.LittleEndian, int64(out.Value))

		binary.Write(&buf, binary.LittleEndian, int64(len(out.PubKeyHash)))
		buf.Write(out.PubKeyHash)
	}

	return buf.Bytes()
}

func DeserializeOutputs(data []byte) TXOutputs {
	var outs TXOutputs
	buf := bytes.NewReader(data)

	var count int64
	err := binary.Read(buf, binary.LittleEndian, &count)
	if err != nil {
		log.Panic(err)
	}

	outs.Outputs = make([]TXOutput, count)

	for i := int64(0); i < count; i++ {
		var val int64
		binary.Read(buf, binary.LittleEndian, &val)

		var keyLen int64
		binary.Read(buf, binary.LittleEndian, &keyLen)

		key := make([]byte, keyLen)
		_, err := buf.Read(key)
		if err != nil {
			log.Panic(err)
		}

		outs.Outputs[i] = TXOutput{
			Value:      int(val),
			PubKeyHash: key,
		}
	}

	return outs
}
