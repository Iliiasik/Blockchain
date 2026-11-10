package core

import "bytes"

type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

func (out *TXOutput) Lock(address []byte) {
	out.PubKeyHash = AddressToPubKeyHash(string(address))
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
	buf := writeInt(nil, int64(len(outs.Outputs)))

	for _, out := range outs.Outputs {
		buf = writeInt(buf, int64(out.Value))
		buf = writeBytes(buf, out.PubKeyHash)
	}

	return buf
}

func DeserializeOutputs(data []byte) TXOutputs {
	var outs TXOutputs
	buf := bytes.NewReader(data)

	count := readInt(buf)
	for i := int64(0); i < count && buf.Len() > 0; i++ {
		var out TXOutput
		out.Value = int(readInt(buf))
		out.PubKeyHash = readBytes(buf)
		outs.Outputs = append(outs.Outputs, out)
	}

	return outs
}
