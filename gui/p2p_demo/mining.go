package p2p_demo

import (
	"Blockchain/core"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (p *P2PDemoUI) forkNode(nodeName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	node := p.nodes[nodeName]
	node.mu.Lock()
	defer node.mu.Unlock()

	if len(node.Blocks) < 2 {
		p.showNotification("Cannot fork: chain is too short")
		return
	}

	fyne.Do(func() {
		p.loadingSpinner.Show()
	})

	go func() {
		fyne.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			prevBlock := node.Blocks[len(node.Blocks)-2]
			tx := core.NewCoinbaseTX("1P2PDemoForkAddress", "Fork on "+nodeName, 16)
			forkBlock := core.NewBlock([]*core.Transaction{tx}, prevBlock.Hash, 10)

			node.Blocks = node.Blocks[:len(node.Blocks)-1]
			node.Blocks = append(node.Blocks, forkBlock)
			p.loadingSpinner.Hide()
			p.showNotification(fmt.Sprintf("Fork created on %s! New chain length: %d", nodeName, len(node.Blocks)))
		})
	}()
}

func (p *P2PDemoUI) resetAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	fyne.Do(func() {
		p.loadingSpinner.Show()
		dialog.ShowConfirm("Reset all nodes",
			"Are you sure you want to reset all nodes to genesis state?",
			func(confirm bool) {
				if confirm {
					go func() {
						fyne.Do(func() {
							p.mu.Lock()
							defer p.mu.Unlock()

							genesisAddress := "1P2PDemoGenesisAddress"
							genesisTx := core.NewCoinbaseTX(genesisAddress, "Genesis block for P2P Demo", 10)
							genesisBlock := core.NewGenesisBlock(genesisTx, 16)

							for _, node := range p.nodes {
								node.mu.Lock()
								node.Blocks = []*core.Block{genesisBlock}
								node.mu.Unlock()
							}
							p.loadingSpinner.Hide()
							p.showNotification("All nodes reset to genesis state")
						})
					}()
				} else {
					p.loadingSpinner.Hide()
				}
			}, p.window)
	})
}

func (p *P2PDemoUI) mineBlock(nodeName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	node := p.nodes[nodeName]
	node.mu.Lock()
	defer node.mu.Unlock()

	fyne.Do(func() {
		p.loadingSpinner.Show()
	})

	go func() {
		var prevHash []byte
		if len(node.Blocks) > 0 {
			lastBlock := node.Blocks[len(node.Blocks)-1]
			prevHash = lastBlock.Hash
		}

		tx := core.NewCoinbaseTX("1P2PDemoAddress", "Mined on "+nodeName, 10)
		block := core.NewBlock([]*core.Transaction{tx}, prevHash, p.localTargetBits)

		fyne.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			node.Blocks = append(node.Blocks, block)
			p.loadingSpinner.Hide()
			p.showNotification(fmt.Sprintf("New block mined on %s! Chain length: %d", nodeName, len(node.Blocks)))
		})
	}()
}
