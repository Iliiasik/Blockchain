package p2p_demo

import (
	"Blockchain/core"
	"bytes"
)

func (p *P2PDemoUI) syncNodes(from, to int) {
	p.mu.Lock()
	source := p.nodes[from]
	target := p.nodes[to]
	result := p.adopt(target, source)
	p.mu.Unlock()

	switch result {
	case adopted:
		p.logf("Synced %s to %s, chain length %d", source.Name, target.Name, len(target.Blocks))
	case invalid:
		p.logf("%s rejected the chain of %s: validation failed", target.Name, source.Name)
	case unchanged:
		p.logf("No sync needed: %s is not longer than %s", source.Name, target.Name)
	}
	p.render()
}

func (p *P2PDemoUI) autoSync() {
	p.mu.Lock()
	var longest *P2PNode
	for _, node := range p.nodes {
		if longest == nil || len(node.Blocks) > len(longest.Blocks) {
			longest = node
		}
	}

	if longest == nil {
		p.mu.Unlock()
		return
	}

	updated := 0
	for _, node := range p.nodes {
		if node != longest && p.adopt(node, longest) == adopted {
			updated++
		}
	}
	length := len(longest.Blocks)
	p.mu.Unlock()

	p.logf("Auto-sync complete, %d nodes switched, all nodes follow chain length %d", updated, length)
	p.render()
}

type syncResult int

const (
	unchanged syncResult = iota
	adopted
	invalid
)

func (p *P2PDemoUI) adopt(target, source *P2PNode) syncResult {
	if target.busy || len(source.Blocks) <= len(target.Blocks) {
		return unchanged
	}
	if !validChain(source.Blocks) {
		return invalid
	}

	newBlocks := make([]*core.Block, len(source.Blocks))
	for i, block := range source.Blocks {
		newBlocks[i] = core.DeserializeBlock(block.Serialize())
	}
	target.Blocks = newBlocks

	return adopted
}

func validChain(blocks []*core.Block) bool {
	for i, block := range blocks {
		if !core.NewProofOfWork(block).Validate() {
			return false
		}
		if i > 0 && !bytes.Equal(block.PrevBlockHash, blocks[i-1].Hash) {
			return false
		}
	}
	return true
}
