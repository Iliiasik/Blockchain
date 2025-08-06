package core

import (
	"crypto/sha256"
)

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleTree(data [][]byte) *MerkleTree {
	if len(data) == 0 {
		return &MerkleTree{}
	}

	var nodes []*MerkleNode

	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, node)
	}

	for len(nodes) > 1 {
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}

		var newLevel []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			node := NewMerkleNode(nodes[i], nodes[i+1], nil)
			newLevel = append(newLevel, node)
		}

		nodes = newLevel
	}

	return &MerkleTree{nodes[0]}
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}
