package mining

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"Blockchain/resources/icons"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MiningUI struct {
	state *state.AppState
	snap  *state.Snapshot

	content  fyne.CanvasObject
	miner    *widget.Select
	subsidy  *widget.Slider
	bitsBar  *widget.Slider
	pool     *fyne.Container
	all      *widget.Check
	picked   map[string]bool
	reward   *ui.Text
	start    *widget.Button
	abort    *widget.Button
	noChain  *ui.Text
	led      *ui.LED
	phase    *ui.Text
	nonce    *ui.Text
	rate     *readout
	elapsed  *readout
	expected *readout
	chance   *readout
	hash     *hexLine
	target   *hexLine
	progress *ui.Meter
	result   *fyne.Container
	log      *fyne.Container
	stop     chan struct{}
	bits     int
}

func New(st *state.AppState) *MiningUI {
	m := &MiningUI{state: st, snap: st.Snapshot(), picked: make(map[string]bool)}
	m.content = m.build()
	st.Subscribe(m.update)
	return m
}

func (m *MiningUI) Content() fyne.CanvasObject {
	return m.content
}

func (m *MiningUI) SetActive(active bool) {
	if active {
		m.subsidy.SetValue(float64(m.state.Subsidy))
		m.bitsBar.SetValue(float64(m.state.TargetBits))
	}
}

func (m *MiningUI) Focus(address string) {
	m.miner.SetSelected(address)
}

func (m *MiningUI) build() fyne.CanvasObject {
	m.miner = widget.NewSelect(nil, func(string) { m.updateControls() })
	m.miner.PlaceHolder = "Wallet that receives the reward"

	subsidyLabel := ui.NewText("", 13, theme.ColorNameForeground)
	subsidy := widget.NewSlider(1, 100)
	subsidy.Step = 1
	subsidy.OnChanged = func(v float64) {
		m.state.Subsidy = int(v)
		subsidyLabel.SetText(fmt.Sprintf("Block reward (subsidy): %d", m.state.Subsidy))
		m.updateReward()
	}

	bitsLabel := ui.NewText("", 13, theme.ColorNameForeground)
	bitsHint := ui.NewText("", 11, resources.ColorNameMuted)
	bits := widget.NewSlider(8, 28)
	bits.Step = 1
	bits.OnChanged = func(v float64) {
		m.state.TargetBits = int(v)
		bitsLabel.SetText(fmt.Sprintf("Mining difficulty (bits): %d", m.state.TargetBits))
		if m.state.TargetBits > 20 {
			bitsHint.SetColor(theme.ColorNameWarning)
			bitsHint.SetText("Above 20 bits mining can take a very long time")
		} else {
			bitsHint.SetColor(resources.ColorNameMuted)
			bitsHint.SetText(fmt.Sprintf("About %s hashes on average", ui.Attempts(m.state.TargetBits)))
		}
		if !m.state.IsMining() {
			m.showTarget(m.state.TargetBits)
		}
	}

	m.all = widget.NewCheck("Include all", func(on bool) {
		for _, tx := range m.snap.Mempool {
			m.picked[hex.EncodeToString(tx.ID)] = on
		}
		m.renderPool()
	})
	m.pool = container.NewVBox()
	poolScroll := ui.NewNestedScroll(m.pool)
	poolScroll.SetMinSize(fyne.NewSize(0, 150))

	m.reward = ui.NewText("", 14, theme.ColorNameForeground)
	m.noChain = ui.Secondary("")
	m.start = ui.Primary("Start mining", theme.MediaPlayIcon(), m.onStart)
	m.abort = ui.Button("Abort", theme.MediaStopIcon(), m.state.CancelMining)
	m.abort.Disable()

	config := ui.NewSurface(container.NewBorder(
		container.NewVBox(
			ui.Section("Next block", "What the block you mine will contain"),
			ui.Padded(8, 0, 0, 0, ui.Caption("Miner")),
			m.miner,
			ui.Padded(8, 0, 0, 0, subsidyLabel), subsidy,
			bitsLabel, bits, bitsHint,
			ui.Padded(8, 8, 0, 0, ui.Divider()),
			container.NewBorder(nil, nil, container.NewCenter(ui.Section("Mempool", "")), m.all),
		),
		container.NewVBox(
			ui.Padded(0, 8, 0, 0, ui.Divider()),
			m.reward,
			m.noChain,
			ui.Padded(8, 0, 0, 0, container.NewGridWithColumns(2, m.abort, m.start)),
		),
		nil, nil,
		poolScroll,
	))
	config.Padding = 20

	m.led = ui.NewLED(16)
	m.led.SetState(ui.LEDOff)
	m.phase = ui.Secondary("Standby")
	m.nonce = ui.Strong("0", 56)
	m.rate = newReadout("Hash rate")
	m.elapsed = newReadout("Elapsed")
	m.expected = newReadout("Expected attempts")
	m.chance = newReadout("Progress")
	m.hash = newHexLine()
	m.target = newHexLine()
	m.progress = ui.NewMeter()
	m.result = container.NewVBox()
	m.log = container.NewVBox()

	readout := ui.NewPanel(container.New(layout.NewCustomPaddedVBoxLayout(8),
		container.NewBorder(nil, nil,
			container.NewCenter(ui.Section("Proof of work", "")),
			container.NewHBox(container.NewCenter(m.led), container.NewCenter(m.phase)),
		),
		ui.Padded(8, 0, 0, 0, ui.Caption("Nonce")),
		m.nonce,
		ui.Padded(0, 8, 0, 0, container.NewGridWithColumns(4, m.rate.box, m.elapsed.box, m.expected.box, m.chance.box)),
		m.progress,
		ui.Padded(8, 0, 0, 0, ui.Caption("Current hash")),
		m.hash.box,
		ui.Caption("Target, the hash must be lower"),
		m.target.box,
		m.result,
	))

	console := ui.NewSurface(container.NewBorder(ui.Section("Console", ""), nil, nil, nil, ui.Padded(8, 0, 0, 0, ui.NewNestedScroll(m.log))))

	subsidy.SetValue(float64(m.state.Subsidy))
	bits.SetValue(float64(m.state.TargetBits))
	m.subsidy = subsidy
	m.bitsBar = bits
	m.showTarget(m.state.TargetBits)
	m.updateReward()
	m.logLine("Miner ready, choose a wallet and start")

	split := container.NewVSplit(ui.NewNestedScroll(ui.Padded(0, 8, 0, 0, readout)), ui.Padded(8, 0, 0, 0, console))
	split.Offset = 0.68

	return container.NewBorder(nil, nil, ui.Padded(0, 0, 0, 16, ui.FixedWidth(400, config)), nil, split)
}

func (m *MiningUI) update(snap *state.Snapshot) {
	m.snap = snap

	current := m.miner.Selected
	m.miner.SetOptions(snap.Addresses())
	if current != "" {
		m.miner.SetSelected(current)
	} else if len(snap.Wallets) > 0 {
		m.miner.SetSelected(snap.Wallets[0].Address)
	}

	alive := make(map[string]bool)
	for _, tx := range snap.Mempool {
		id := hex.EncodeToString(tx.ID)
		alive[id] = true
	}
	for id := range m.picked {
		if !alive[id] {
			delete(m.picked, id)
		}
	}

	m.renderPool()
	m.updateControls()
}

func (m *MiningUI) renderPool() {
	m.pool.RemoveAll()
	if len(m.snap.Mempool) == 0 {
		m.pool.Add(ui.Muted("The mempool is empty. The block will contain only the coinbase transaction."))
	}

	for _, tx := range m.snap.Mempool {
		id := hex.EncodeToString(tx.ID)
		check := widget.NewCheck("", func(on bool) {
			m.picked[id] = on
			m.updateReward()
		})
		check.SetChecked(m.picked[id])

		row := ui.NewRaised(container.NewBorder(nil, nil, check,
			container.NewCenter(ui.Secondary(fmt.Sprintf("Fee %d", tx.Fee))),
			container.NewVBox(
				ui.Mono(ui.ShortString(id, 10, 6), theme.ColorNameForeground),
				ui.Caption(fmt.Sprintf("%d to %s", tx.GetAmount(), ui.ShortAddress(tx.GetRecipientAddress()))),
			),
		))
		row.Padding = 8
		m.pool.Add(row)
	}
	m.pool.Refresh()

	m.updateReward()
}

func (m *MiningUI) selectedTransactions() []*core.Transaction {
	var txs []*core.Transaction
	for _, tx := range m.snap.Mempool {
		if m.picked[hex.EncodeToString(tx.ID)] {
			txs = append(txs, tx)
		}
	}
	return txs
}

func (m *MiningUI) updateReward() {
	if m.reward == nil {
		return
	}
	fees := 0
	txs := m.selectedTransactions()
	for _, tx := range txs {
		fees += tx.Fee
	}
	m.reward.SetText(fmt.Sprintf("Reward %d, subsidy %d plus fees %d, %d tx", m.state.Subsidy+fees, m.state.Subsidy, fees, len(txs)+1))
}

func (m *MiningUI) updateControls() {
	mining := m.state.IsMining()

	switch {
	case !m.snap.HasChain:
		m.noChain.SetText("Create the blockchain first.")
	case m.miner.Selected == "":
		m.noChain.SetText("Create a wallet to receive the reward.")
	default:
		m.noChain.SetText("")
	}

	if mining || !m.snap.HasChain || m.miner.Selected == "" {
		m.start.Disable()
	} else {
		m.start.Enable()
	}
	if mining {
		m.abort.Enable()
	} else {
		m.abort.Disable()
	}
}

func (m *MiningUI) onStart() {
	miner := m.miner.Selected
	txs := m.selectedTransactions()
	m.bits = m.state.TargetBits
	height := m.snap.Height()

	m.result.RemoveAll()
	m.result.Refresh()
	m.showTarget(m.bits)
	m.setPhase("Searching", ui.LEDBusy)
	m.logLine(fmt.Sprintf("Mining block #%d with %d tx at %d bits", height, len(txs)+1, m.bits))

	m.state.Mine(miner, txs, func(block *core.Block, err error) {
		m.stopTicker()
		m.tick()
		m.updateControls()

		switch {
		case errors.Is(err, core.ErrMiningCancelled):
			m.setPhase("Aborted", ui.LEDAlert)
			m.logLine("Mining aborted by user")
		case err != nil:
			m.setPhase("Rejected", ui.LEDAlert)
			m.logLine("Error: " + err.Error())
			m.state.Toast("Mining failed", err.Error(), state.LevelError)
		default:
			m.setPhase("Block found", ui.LEDIdle)
			m.logLine(fmt.Sprintf("Block #%d accepted, nonce %d", height, block.Nonce))
			m.showResult(block, height)
			m.state.Toast("Block mined", fmt.Sprintf("Block #%d, %d transactions", height, len(block.Transactions)), state.LevelSuccess)
		}
	})

	m.updateControls()
	if !m.state.IsMining() {
		return
	}

	m.stop = make(chan struct{})
	go func(stop chan struct{}) {
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fyne.Do(m.tick)
			}
		}
	}(m.stop)
}

func (m *MiningUI) stopTicker() {
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
}

func (m *MiningUI) tick() {
	nonce, hash, elapsed := m.state.Progress.Snapshot()
	expected := float64(int64(1) << uint(m.bits))

	m.nonce.SetText(ui.Number(nonce))
	m.hash.Set(hex.EncodeToString(hash[:]), m.bits, true)
	m.elapsed.Set(formatDuration(elapsed))

	rate := 0.0
	if elapsed > 0 {
		rate = float64(nonce) / elapsed.Seconds()
	}
	m.rate.Set(formatRate(rate))

	ratio := float64(nonce) / expected
	m.chance.Set(fmt.Sprintf("%.0f%%", 100*ratio))
	m.progress.SetValue(ratio)
}

func (m *MiningUI) showTarget(bits int) {
	m.bits = bits
	m.expected.Set(ui.Attempts(bits))
	m.target.Set(ui.Target(bits), bits, false)
}

func (m *MiningUI) showResult(block *core.Block, height int) {
	inspect := ui.Button("Inspect block", theme.SearchIcon(), func() { m.state.Go("blockchain", "") })

	panel := ui.NewRaised(container.NewVBox(
		container.NewBorder(nil, nil,
			container.NewHBox(ui.Icon(icons.ResourceCoinPng, 24), container.NewCenter(ui.NewText(fmt.Sprintf("Block #%d accepted", height), 16, theme.ColorNameForeground))),
			inspect,
		),
		ui.KV(
			ui.Key("Hash"), ui.MonoValue(hex.EncodeToString(block.Hash)),
			ui.Key("Nonce"), ui.Value(ui.Number(int64(block.Nonce))),
			ui.Key("Reward"), ui.Value(fmt.Sprintf("%d to %s", block.Transactions[0].GetAmount(), ui.ShortAddress(block.Transactions[0].GetRecipientAddress()))),
		),
	))
	m.result.Objects = []fyne.CanvasObject{panel}
	m.result.Refresh()
}

func (m *MiningUI) setPhase(text string, led ui.LEDState) {
	m.phase.SetText(text)
	m.led.SetState(led)
}

type readout struct {
	value *ui.Text
	box   fyne.CanvasObject
}

func newReadout(label string) *readout {
	r := &readout{value: ui.Strong("0", 20)}
	r.box = container.NewVBox(ui.Caption(label), r.value)
	return r
}

func (r *readout) Set(value string) {
	r.value.SetText(value)
}

type hexLine struct {
	head *ui.Text
	tail *ui.Text
	box  fyne.CanvasObject
}

func newHexLine() *hexLine {
	h := &hexLine{head: ui.Mono("", theme.ColorNameForeground), tail: ui.Mono("", resources.ColorNameMuted)}
	h.box = container.New(layout.NewCustomPaddedHBoxLayout(0), h.head, h.tail)
	return h
}

func (h *hexLine) Set(value string, bits int, check bool) {
	digits := min((bits+3)/4, len(value))
	head := value[:digits]

	h.head.SetColor(theme.ColorNameForeground)
	if check {
		h.head.SetColor(resources.ColorNameMuted)
		if meetsTarget(value, bits) {
			h.head.SetColor(theme.ColorNameSuccess)
		}
	}

	h.head.SetText(head)
	h.tail.SetText(value[digits:])
}

func (m *MiningUI) logLine(text string) {
	line := ui.Mono(time.Now().Format("15:04:05")+"  "+text, resources.ColorNameMuted)
	m.log.Objects = append([]fyne.CanvasObject{line}, m.log.Objects...)
	if len(m.log.Objects) > 60 {
		m.log.Objects = m.log.Objects[:60]
	}
	m.log.Refresh()
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1f s", d.Seconds())
}

func formatRate(rate float64) string {
	switch {
	case rate >= 1e6:
		return fmt.Sprintf("%.2f MH/s", rate/1e6)
	case rate >= 1e3:
		return fmt.Sprintf("%.1f kH/s", rate/1e3)
	}
	return fmt.Sprintf("%.0f H/s", rate)
}

func meetsTarget(hash string, bits int) bool {
	value, ok := new(big.Int).SetString(hash, 16)
	return ok && value.BitLen() <= 256-bits
}
