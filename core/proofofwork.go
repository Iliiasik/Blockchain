package core

import (
	"Blockchain/utils"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync/atomic"
	"time"
)

const progressInterval = 1 << 12

var (
	maxNonce = math.MaxInt64

	ErrMiningCancelled = errors.New("mining cancelled")
)

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

type MiningProgress struct {
	nonce   atomic.Int64
	hash    atomic.Value
	started atomic.Int64
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-b.Bits))

	pow := &ProofOfWork{b, target}
	return pow
}

func (pow *ProofOfWork) prepareHeader() []byte {
	return bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			pow.block.HashTransactions(),
			utils.IntToHex(pow.block.Timestamp),
			utils.IntToHex(int64(pow.block.Bits)),
		},
		[]byte{},
	)
}

func (pow *ProofOfWork) prepareData(nonce int) []byte {
	return append(pow.prepareHeader(), utils.IntToHex(int64(nonce))...)
}

func (pow *ProofOfWork) Run(ctx context.Context, progress *MiningProgress) (int, []byte, error) {
	header := pow.prepareHeader()
	data := make([]byte, len(header)+8)
	copy(data, header)
	nonceBytes := data[len(header):]

	progress.reset()
	fmt.Printf("Mining a new block with %d bits\n", pow.block.Bits)

	for nonce := 0; nonce < maxNonce; nonce++ {
		binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
		hash := sha256.Sum256(data)

		if hasLeadingZeroBits(hash, pow.block.Bits) {
			progress.report(nonce, hash)
			fmt.Printf("%x\n\n", hash)
			return nonce, hash[:], nil
		}

		if nonce%progressInterval == 0 {
			progress.report(nonce, hash)
			if ctx.Err() != nil {
				return 0, nil, ErrMiningCancelled
			}
		}
	}

	return 0, nil, errors.New("nonce range exhausted")
}

func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	return hashInt.Cmp(pow.target) == -1
}

func hasLeadingZeroBits(hash [32]byte, bits int) bool {
	full := bits / 8
	for i := 0; i < full; i++ {
		if hash[i] != 0 {
			return false
		}
	}

	rest := bits % 8
	return rest == 0 || hash[full]>>(8-rest) == 0
}

func (p *MiningProgress) reset() {
	if p == nil {
		return
	}
	p.nonce.Store(0)
	p.hash.Store([32]byte{})
	p.started.Store(time.Now().UnixNano())
}

func (p *MiningProgress) report(nonce int, hash [32]byte) {
	if p == nil {
		return
	}
	p.nonce.Store(int64(nonce))
	p.hash.Store(hash)
}

func (p *MiningProgress) Snapshot() (int64, [32]byte, time.Duration) {
	hash, _ := p.hash.Load().([32]byte)
	started := p.started.Load()
	if started == 0 {
		return p.nonce.Load(), hash, 0
	}

	return p.nonce.Load(), hash, time.Since(time.Unix(0, started))
}
