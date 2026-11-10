package core

import bolt "go.etcd.io/bbolt"

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return nil
		}
		if encodedBlock := b.Get(i.currentHash); encodedBlock != nil {
			block = DeserializeBlock(encodedBlock)
		}

		return nil
	})

	if err != nil || block == nil {
		return nil
	}

	i.currentHash = block.PrevBlockHash

	return block
}
