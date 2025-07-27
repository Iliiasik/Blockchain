package p2p_demo

import (
	"Blockchain/core"
	"fmt"
	"fyne.io/fyne/v2"
)

func (p *P2PDemoUI) syncNodes(from, to string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	source := p.nodes[from]
	target := p.nodes[to]

	source.mu.Lock()
	target.mu.Lock()
	defer source.mu.Unlock()
	defer target.mu.Unlock()

	fyne.Do(func() {
		p.loadingSpinner.Show()
	})

	go func() {
		fyne.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			if len(source.Blocks) > len(target.Blocks) {
				newBlocks := make([]*core.Block, len(source.Blocks))
				for i, block := range source.Blocks {
					blockCopy := core.DeserializeBlock(block.Serialize())
					newBlocks[i] = blockCopy
				}
				target.Blocks = newBlocks
				p.showNotification(fmt.Sprintf("Synced %s → %s. New chain length: %d", from, to, len(target.Blocks)))
			} else {
				p.showNotification(fmt.Sprintf("No sync needed: %s is not longer than %s", from, to))
			}
			p.loadingSpinner.Hide()
		})
	}()
}

func (p *P2PDemoUI) autoSync() {
	p.mu.Lock()
	defer p.mu.Unlock()

	fyne.Do(func() {
		p.loadingSpinner.Show()
	})

	go func() {

		fyne.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			var longestNode *P2PNode
			maxLength := 0

			for _, node := range p.nodes {
				node.mu.Lock()
				if len(node.Blocks) > maxLength {
					maxLength = len(node.Blocks)
					longestNode = node
				}
				node.mu.Unlock()
			}

			if longestNode != nil {
				for _, node := range p.nodes {
					if node != longestNode {
						node.mu.Lock()
						newBlocks := make([]*core.Block, len(longestNode.Blocks))
						for i, block := range longestNode.Blocks {
							blockCopy := core.DeserializeBlock(block.Serialize())
							newBlocks[i] = blockCopy
						}
						node.Blocks = newBlocks
						node.mu.Unlock()
					}
				}
				p.showNotification(fmt.Sprintf("Auto-sync complete. All nodes now have chain length: %d", maxLength))
			} else {
				p.showNotification("Auto-sync failed: no nodes found")
			}
			p.loadingSpinner.Hide()
		})
	}()
}
