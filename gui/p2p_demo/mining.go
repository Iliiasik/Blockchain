package p2p_demo

import (
	"Blockchain/core"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (p *P2PDemoUI) mineBlock(node *P2PNode) {
	p.mu.Lock()
	if node.busy {
		p.mu.Unlock()
		return
	}
	node.busy = true
	prevHash := node.Blocks[len(node.Blocks)-1].Hash
	bits := p.localTargetBits
	p.mu.Unlock()

	p.render()
	p.logf("%s started mining on top of %x", node.Name, prevHash[:6])

	go func() {
		tx := core.NewCoinbaseTX(node.Miner, "Mined on "+node.Name, 10)
		block, err := core.NewBlock([]*core.Transaction{tx}, prevHash, bits)

		p.mu.Lock()
		node.busy = false
		if err != nil {
			p.mu.Unlock()
			fyne.Do(func() {
				p.logf("%s could not mine a block: %v", node.Name, err)
				p.render()
			})
			return
		}
		tip := node.Blocks[len(node.Blocks)-1].Hash
		accepted := string(tip) == string(prevHash)
		if accepted {
			node.Blocks = append(node.Blocks, block)
		}
		length := len(node.Blocks)
		p.mu.Unlock()

		fyne.Do(func() {
			if accepted {
				p.logf("New block mined on %s, chain length %d", node.Name, length)
			} else {
				p.logf("%s discarded a stale block, its chain changed while mining", node.Name)
			}
			p.render()
		})
	}()
}

func (p *P2PDemoUI) forkNode(node *P2PNode) {
	p.mu.Lock()
	if node.busy || len(node.Blocks) < 2 {
		p.mu.Unlock()
		p.logf("Cannot fork %s: chain is too short", node.Name)
		return
	}
	node.busy = true
	prevBlock := node.Blocks[len(node.Blocks)-2]
	bits := p.localTargetBits
	p.mu.Unlock()

	p.render()

	go func() {
		tx := core.NewCoinbaseTX(node.Miner, "Fork on "+node.Name, 16)
		forkBlock, err := core.NewBlock([]*core.Transaction{tx}, prevBlock.Hash, bits)

		p.mu.Lock()
		node.busy = false
		if err != nil {
			p.mu.Unlock()
			fyne.Do(func() {
				p.logf("%s could not mine a fork block: %v", node.Name, err)
				p.render()
			})
			return
		}
		if string(node.Blocks[len(node.Blocks)-2].Hash) == string(prevBlock.Hash) {
			node.Blocks = append(node.Blocks[:len(node.Blocks)-1:len(node.Blocks)-1], forkBlock)
		}
		length := len(node.Blocks)
		p.mu.Unlock()

		fyne.Do(func() {
			p.logf("Fork created on %s, chain length %d", node.Name, length)
			p.render()
		})
	}()
}

func (p *P2PDemoUI) resetAll() {
	confirm := dialog.NewConfirm("Reset all nodes",
		"Are you sure you want to reset all nodes to genesis state?",
		func(ok bool) {
			if !ok {
				return
			}

			go func() {
				genesis, err := p.newGenesis()
				if err != nil {
					fyne.Do(func() { p.logf("Reset failed: %v", err) })
					return
				}

				p.mu.Lock()
				for _, node := range p.nodes {
					node.Blocks = []*core.Block{genesis}
				}
				p.mu.Unlock()

				fyne.Do(func() {
					p.logf("All nodes reset to genesis state %s", fmt.Sprintf("%x", genesis.Hash[:6]))
					p.render()
				})
			}()
		}, p.window)
	confirm.SetConfirmImportance(widget.DangerImportance)
	confirm.Show()
}
