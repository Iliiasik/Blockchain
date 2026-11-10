package utils

import "encoding/binary"

func IntToHex(num int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(num))

	return buf
}

func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
