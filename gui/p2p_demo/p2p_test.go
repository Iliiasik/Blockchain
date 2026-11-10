package p2p_demo

import (
	"Blockchain/core"
	"Blockchain/resources"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func chain(t *testing.T, length int) []*core.Block {
	t.Helper()
	address := string(core.NewWallet().GetAddress())
	genesis, err := core.NewGenesisBlock(core.NewCoinbaseTX(address, "genesis", 10), 8)
	if err != nil {
		t.Fatal(err)
	}
	blocks := []*core.Block{genesis}
	for len(blocks) < length {
		block, err := core.NewBlock([]*core.Transaction{core.NewCoinbaseTX(address, "", 10)}, blocks[len(blocks)-1].Hash, 8)
		if err != nil {
			t.Fatal(err)
		}
		blocks = append(blocks, block)
	}
	return blocks
}

func TestValidChain(t *testing.T) {
	blocks := chain(t, 4)
	if !validChain(blocks) {
		t.Fatal("honest chain rejected")
	}

	broken := append([]*core.Block{}, blocks...)
	relinked := *broken[2]
	relinked.PrevBlockHash = broken[0].Hash
	broken[2] = &relinked
	if validChain(broken) {
		t.Fatal("chain with a broken link accepted")
	}

	forged := append([]*core.Block{}, blocks...)
	changed := *forged[1]
	changed.Nonce++
	forged[1] = &changed
	if !core.NewProofOfWork(&changed).Validate() && validChain(forged) {
		t.Fatal("chain with invalid proof of work accepted")
	}
}

func TestAdoptFollowsLongestValidChain(t *testing.T) {
	p := &P2PDemoUI{}
	long := chain(t, 3)
	target := &P2PNode{Name: "A", Blocks: long[:1]}
	source := &P2PNode{Name: "B", Blocks: long}

	if got := p.adopt(target, source); got != adopted {
		t.Fatalf("adopt = %v, want adopted", got)
	}
	if len(target.Blocks) != 3 || target.Blocks[2] == source.Blocks[2] {
		t.Fatal("the adopted chain must be a deep copy")
	}
	if got := p.adopt(target, source); got != unchanged {
		t.Fatal("an equal chain must not be adopted")
	}
	if got := p.adopt(source, &P2PNode{Blocks: long[:2]}); got != unchanged {
		t.Fatal("a shorter chain must not be adopted")
	}

	busy := &P2PNode{Blocks: long[:1], busy: true}
	if got := p.adopt(busy, source); got != unchanged {
		t.Fatal("a mining node must not switch chains")
	}

	bad := append([]*core.Block{}, chain(t, 4)...)
	relinked := *bad[3]
	relinked.PrevBlockHash = bad[1].Hash
	bad[3] = &relinked
	if got := p.adopt(target, &P2PNode{Blocks: bad}); got != invalid || len(target.Blocks) != 3 {
		t.Fatal("an invalid longer chain must be rejected")
	}
}

func TestNetworkScreenSync(t *testing.T) {
	test.NewTempApp(t).Settings().SetTheme(&resources.Theme{Dark: true})
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1200, 800))

	p := New(w)
	w.SetContent(p.content)
	blocks := chain(t, 3)
	p.nodes = []*P2PNode{
		{Name: "Node A", Blocks: blocks},
		{Name: "Node B", Blocks: blocks[:1]},
		{Name: "Node C", Blocks: blocks[:2]},
	}
	p.content.Objects = []fyne.CanvasObject{p.buildNetwork()}
	p.content.Refresh()

	for _, node := range p.nodes {
		if node.lane == nil {
			t.Fatalf("%s has no lane", node.Name)
		}
	}

	p.syncNodes(0, 1)
	if len(p.nodes[1].Blocks) != 3 {
		t.Fatal("B did not adopt the longer chain of A")
	}
	p.syncNodes(1, 0)
	if len(p.log.Objects) != 2 {
		t.Fatalf("log has %d lines", len(p.log.Objects))
	}

	p.autoSync()
	for _, node := range p.nodes {
		if len(node.Blocks) != 3 {
			t.Fatalf("%s follows %d blocks after auto sync", node.Name, len(node.Blocks))
		}
	}

	for i := 0; i < 100; i++ {
		p.logf("line %d", i)
	}
	if len(p.log.Objects) != 80 {
		t.Fatalf("log keeps %d lines, want the newest 80", len(p.log.Objects))
	}
}
