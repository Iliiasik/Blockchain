package visual

import (
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/color"
	"math"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func demoID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func demoHash(seed string) string {
	return "0000" + demoID(seed)[4:]
}

var (
	demoCoinbase = []string{demoID("cb0"), demoID("cb1"), demoID("cb2"), demoID("cb3")}
	demoAlice    = demoID("t1")
	demoBob      = demoID("t2")
	demoPending  = demoID("t3")
)

func demoModel(mined bool) *model {
	m := newModel()
	m.maxPool = 10
	coinbase := func(height, value int, to string) *txn {
		return &txn{id: demoCoinbase[height], block: height, coinbase: true, outputs: []output{{value: value, owner: to}}}
	}

	t1 := &txn{id: demoAlice, block: 1,
		inputs:  []input{{txid: demoCoinbase[0], vout: 0, owner: "alice"}},
		outputs: []output{{value: 4, owner: "bob"}, {value: 6, owner: "alice"}}}
	t2 := &txn{id: demoBob, block: 2, fee: 1,
		inputs:  []input{{txid: demoAlice, vout: 0, owner: "bob"}},
		outputs: []output{{value: 3, owner: "alice"}}}
	t3 := &txn{id: demoPending, block: -1, fee: 1,
		inputs:  []input{{txid: demoAlice, vout: 1, owner: "alice"}},
		outputs: []output{{value: 5, owner: "bob"}}}

	blocks := [][]*txn{
		{coinbase(0, 10, "alice")},
		{coinbase(1, 10, "miner"), t1},
		{coinbase(2, 11, "miner"), t2},
	}
	nonces := []int{40211, 91873, 18342, 67120}
	if mined {
		t3.block = 3
		blocks = append(blocks, []*txn{coinbase(3, 11, "miner"), t3})
	} else {
		m.mempool = []*txn{t3}
	}

	for h, txs := range blocks {
		m.blocks = append(m.blocks, &blockInfo{height: h, hash: demoHash(fmt.Sprint("block", h)), nonce: nonces[h], bits: 16, txs: txs})
		for _, t := range txs {
			m.add(t)
		}
	}
	for _, t := range m.mempool {
		m.add(t)
	}

	balances := map[string]int{"alice": 9, "bob": 0, "miner": 21}
	if mined {
		balances = map[string]int{"alice": 3, "bob": 5, "miner": 32}
	}
	for _, w := range []struct{ id, name string }{{"alice", "Alice"}, {"bob", "Bob"}, {"miner", "Miner"}} {
		m.wallets = append(m.wallets, &walletInfo{address: w.id, name: w.name, named: true, balance: balances[w.id]})
	}

	m.link()
	return m
}

type guideStep struct {
	title  string
	text   string
	focus  []ref
	links  [3]bool
	view   func(g *guide) camera
	mining bool
	spin   bool
}

type guide struct {
	scene   *Scene
	steps   []guideStep
	index   int
	mined   bool
	gen     atomic.Int64
	counter *ui.Text
	title   *ui.Text
	text    *widget.Label
	dots    []*canvas.Circle
	back    *widget.Button
	next    *widget.Button
	replay  *widget.Button
	dialog  dialog.Dialog
}

func showGuide(window fyne.Window) {
	g := &guide{scene: NewScene()}
	g.scene.SetModel(demoModel(false), false)
	g.steps = g.script()

	g.counter = ui.Caption("")
	g.title = ui.Title("", 20)
	g.text = widget.NewLabel("")
	g.text.Wrapping = fyne.TextWrapWord

	dots := container.NewHBox()
	for range g.steps {
		dot := canvas.NewCircle(ui.Color(resources.ColorNameBorder))
		g.dots = append(g.dots, dot)
		dots.Add(container.NewGridWrap(fyne.NewSize(8, 8), dot))
	}

	g.back = ui.Button("Back", theme.NavigateBackIcon(), func() { g.show(g.index - 1) })
	g.next = ui.Primary("Next", theme.NavigateNextIcon(), func() {
		if g.index == len(g.steps)-1 {
			g.dialog.Hide()
			return
		}
		g.show(g.index + 1)
	})
	g.next.IconPlacement = widget.ButtonIconTrailingText
	g.replay = ui.Button("Replay", theme.MediaReplayIcon(), func() { g.mine() })

	legend := container.NewHBox(
		ui.NewBadge("Block", resources.ColorNameMuted),
		ui.NewBadge("Tip", theme.ColorNamePrimary),
		ui.NewBadge("Coinbase", theme.ColorNameSuccess),
		ui.NewBadge("Transfer", resources.ColorNameAccent),
		ui.NewBadge("Pending", theme.ColorNameWarning),
	)

	view := container.NewScroll(g.scene)
	view.Direction = container.ScrollNone
	stage := ui.NewSurface(view)
	stage.Padding = 1
	sizer := canvas.NewRectangle(color.Transparent)
	sizer.SetMinSize(fyne.NewSize(860, 430))

	textBox := canvas.NewRectangle(color.Transparent)
	textBox.SetMinSize(fyne.NewSize(0, 92))

	header := container.NewBorder(nil, nil, container.NewVBox(g.counter, g.title), container.NewCenter(legend))
	footer := container.NewVBox(
		container.NewStack(textBox, g.text),
		container.NewBorder(nil, nil, container.NewCenter(dots), container.NewHBox(g.back, g.replay, g.next)),
	)
	content := container.NewBorder(ui.Padded(0, 12, 0, 0, header), ui.Padded(8, 0, 0, 0, footer), nil, nil, container.NewStack(sizer, stage))

	g.dialog = dialog.NewCustom("How to read the visualizer", "Close", content, window)
	g.dialog.SetOnClosed(func() {
		g.gen.Add(1)
		g.scene.Stop()
	})
	g.dialog.Resize(fyne.NewSize(940, 700))
	g.dialog.Show()

	time.AfterFunc(120*time.Millisecond, func() { fyne.Do(func() { g.show(0) }) })
}

func (g *guide) script() []guideStep {
	blocks := []ref{blockRef(0), blockRef(1), blockRef(2)}
	near := func(target ref, yaw, pitch, dist float64) func(g *guide) camera {
		return func(g *guide) camera {
			at, ok := g.scene.anchor(target)
			if !ok {
				return g.scene.frame()
			}
			at.y = 0.6
			return camera{yaw: yaw, pitch: pitch, dist: dist, target: at}
		}
	}
	between := func(a, b ref, yaw, pitch, dist float64) func(g *guide) camera {
		return func(g *guide) camera {
			pa, ok := g.scene.anchor(a)
			pb, ok2 := g.scene.anchor(b)
			if !ok || !ok2 {
				return g.scene.frame()
			}
			at := lerp3(pa, pb, 0.5)
			at.y = 0.6
			return camera{yaw: yaw, pitch: pitch, dist: dist, target: at}
		}
	}
	overview := func(yaw, pitch float64) func(g *guide) camera {
		return func(g *guide) camera {
			c := g.scene.frame()
			c.yaw, c.pitch = yaw, pitch
			return c
		}
	}

	return []guideStep{
		{
			title: "The chain",
			text:  "Each grey slab is a block. They sit on one line from the oldest on the left to the newest on the right, and the number above a block is its height. #0 is the genesis block, the one with the blue top is the tip.",
			focus: blocks,
			links: [3]bool{true, false, false},
			view:  between(blockRef(0), blockRef(2), 0.18, 0.42, 13),
		},
		{
			title: "Hash links",
			text:  "Every block header stores the hash of the block before it, the arrows point from a block to its parent. Edit an old block and its hash changes, so the arrow after it no longer matches. That is why history cannot be rewritten quietly.",
			focus: blocks,
			links: [3]bool{true, false, false},
			view:  between(blockRef(1), blockRef(2), -0.15, 0.22, 8.5),
		},
		{
			title: "Transactions live inside blocks",
			text:  "The small cubes on top of a block are its transactions. The green one is the coinbase, the first transaction of every block, which creates new coins for the miner. Blue cubes are transfers between wallets. Block #1 holds one of each.",
			focus: []ref{blockRef(1), txRef(demoCoinbase[1]), txRef(demoAlice)},
			view:  near(blockRef(1), 0.35, 0.75, 6.5),
		},
		{
			title: "Where coins come from",
			text:  "A transfer never edits a balance. Each input points at an output of an earlier transaction and spends it whole. The blue arc runs from the spent output to the transaction that spends it: Alice spends the 10 coins she got in #0 and sends 4 to Bob, then Bob spends those 4 in #2.",
			focus: []ref{txRef(demoCoinbase[0]), txRef(demoAlice), txRef(demoBob)},
			links: [3]bool{false, true, false},
			view:  between(blockRef(0), blockRef(2), 0.1, 0.62, 11),
		},
		{
			title: "Wallets and balances",
			text:  "Wallets sit in the front row. A balance is not stored anywhere: it is the sum of the unspent outputs locked to the wallet's key. The green lines show exactly which outputs each wallet owns, and the column height is their total.",
			focus: []ref{walletRef("alice"), walletRef("bob"), walletRef("miner"), txRef(demoBob), txRef(demoCoinbase[1]), txRef(demoCoinbase[2])},
			links: [3]bool{false, false, true},
			view:  overview(0.05, 0.82),
		},
		{
			title: "The mempool",
			text:  "A new transfer does not enter the chain right away. It waits in the mempool as an amber cube. The dashed amber arc shows the output it is going to spend: Alice's 6 coins of change. That output is now reserved, so the same coins cannot be spent twice.",
			focus: []ref{txRef(demoPending), txRef(demoAlice), walletRef("alice"), walletRef("bob")},
			links: [3]bool{false, true, false},
			view:  overview(0.62, 0.62),
		},
		{
			title:  "Mining a block",
			text:   "A miner takes the pending transfers, adds a coinbase paying the subsidy plus all fees, and tries nonce after nonce until the block hash starts with enough zero bits. Then the block joins the chain, the mempool empties and the coins fly to their new owners.",
			links:  [3]bool{true, true, false},
			view:   overview(0.42, 0.5),
			mining: true,
		},
		{
			title: "Find your way around",
			text:  "Drag to rotate, right drag or Shift drag to pan, scroll to zoom. Click any block, transaction or wallet: everything connected to it lights up and the rest fades out, and a card explains what you picked. The switches under the view show or hide each kind of link.",
			links: [3]bool{true, true, true},
			view:  overview(0.42, 0.5),
			spin:  true,
		},
	}
}

func (g *guide) show(index int) {
	if index < 0 || index >= len(g.steps) {
		return
	}
	g.index = index
	step := g.steps[index]
	gen := g.gen.Add(1)

	g.scene.SetSpin(false)
	if step.mining {
		g.mine()
	} else {
		interrupted := g.scene.mining
		g.scene.SetMining(false)
		wantMined := index > 6
		if wantMined != g.mined || interrupted {
			g.mined = wantMined
			g.scene.SetModel(demoModel(wantMined), false)
		}
	}

	g.scene.SetLinkVisible(linkHash, step.links[0])
	g.scene.SetLinkVisible(linkSpend, step.links[1])
	g.scene.SetLinkVisible(linkOwn, step.links[2])
	g.scene.SetFocus(step.focus...)
	g.scene.LookAt(step.view(g), true)
	if step.spin {
		time.AfterFunc(900*time.Millisecond, func() {
			fyne.Do(func() {
				if g.gen.Load() == gen {
					g.scene.SetSpin(true)
				}
			})
		})
	}

	g.counter.SetText(fmt.Sprintf("Step %d of %d", index+1, len(g.steps)))
	g.title.SetText(step.title)
	g.text.SetText(step.text)
	for i, dot := range g.dots {
		dot.FillColor = ui.Color(resources.ColorNameBorder)
		if i == index {
			dot.FillColor = ui.Color(theme.ColorNamePrimary)
		}
		dot.Refresh()
	}

	if index == 0 {
		g.back.Disable()
	} else {
		g.back.Enable()
	}
	if step.mining {
		g.replay.Show()
	} else {
		g.replay.Hide()
	}
	if index == len(g.steps)-1 {
		g.next.SetText("Done")
		g.next.SetIcon(theme.ConfirmIcon())
	} else {
		g.next.SetText("Next")
		g.next.SetIcon(theme.NavigateNextIcon())
	}
}

func (g *guide) mine() {
	gen := g.gen.Add(1)
	g.mined = false
	g.scene.SetModel(demoModel(false), false)
	g.scene.SetFocus()
	g.scene.SetMining(true)

	go func() {
		start := time.Now()
		nonce := 0
		for time.Since(start) < 2600*time.Millisecond {
			time.Sleep(70 * time.Millisecond)
			if g.gen.Load() != gen {
				return
			}
			nonce += 731 + int(math.Mod(float64(nonce)*7.3, 97))
			label := fmt.Sprintf("nonce %s", ui.Number(int64(nonce)))
			fyne.Do(func() {
				if g.gen.Load() == gen {
					g.scene.SetNonce(label)
				}
			})
		}
		fyne.Do(func() {
			if g.gen.Load() != gen {
				return
			}
			g.mined = true
			g.scene.SetMining(false)
			g.scene.SetModel(demoModel(true), true)
		})
	}()
}
