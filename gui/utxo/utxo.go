package utxo

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"encoding/hex"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	allOutputs = "All outputs"
	maxRows    = 300
)

type UTXOTab struct {
	state *state.AppState
	snap  *state.Snapshot

	content fyne.CanvasObject
	filter  *widget.SelectEntry
	outputs []core.UnspentOutput
	rows    *fyne.Container
	entries *ui.Stat
	shown   *ui.Stat
	value   *ui.Stat
	pending *ui.Stat
}

func New(st *state.AppState) *UTXOTab {
	u := &UTXOTab{state: st, snap: st.Snapshot()}
	u.content = u.build()
	st.Subscribe(u.update)
	return u
}

func (u *UTXOTab) Content() fyne.CanvasObject {
	return u.content
}

func (u *UTXOTab) Focus(address string) {
	u.filter.SetText(address)
}

func (u *UTXOTab) build() fyne.CanvasObject {
	u.filter = widget.NewSelectEntry(nil)
	u.filter.SetPlaceHolder("Filter by address or choose a wallet")
	u.filter.OnChanged = func(string) { u.load() }

	reindex := ui.Button("Reindex", theme.ViewRefreshIcon(), u.onReindex)

	u.entries = ui.NewStat("Indexed transactions")
	u.shown = ui.NewStat("Outputs shown")
	u.value = ui.NewStat("Value")
	u.pending = ui.NewStat("Locked by mempool")

	u.rows = container.New(layout.NewCustomPaddedVBoxLayout(8))

	header := ui.NewSurface(container.NewVBox(
		ui.Section("Unspent transaction outputs", "The UTXO set is a cache of every output that has not been spent yet. Balances are computed from it instead of scanning the whole chain."),
		ui.Padded(8, 0, 0, 0, container.NewBorder(nil, nil, nil, reindex, u.filter)),
	))
	header.Padding = 20

	return container.NewBorder(
		container.New(layout.NewCustomPaddedVBoxLayout(16),
			header,
			container.NewGridWithColumns(4, u.entries.Box, u.shown.Box, u.value.Box, u.pending.Box),
			layout.NewSpacer(),
		),
		nil, nil, nil,
		ui.NewNestedScroll(u.rows),
	)
}

func (u *UTXOTab) row(out core.UnspentOutput) fyne.CanvasObject {
	badge := ui.NewBadge("Spendable", theme.ColorNameSuccess)
	if out.Pending {
		badge.Set("In mempool", theme.ColorNameWarning)
	}
	return ui.NewRaised(container.NewBorder(nil, nil, nil,
		container.NewHBox(
			ui.FixedWidth(120, container.NewCenter(badge)),
			ui.FixedWidth(64, container.NewCenter(ui.NewText(ui.Number(int64(out.Value)), 16, theme.ColorNameForeground))),
		),
		container.NewVBox(
			ui.Mono(fmt.Sprintf("%s : %d", ui.ShortString(hex.EncodeToString(out.TxID), 16, 8), out.Index), theme.ColorNameForeground),
			ui.Caption("Locked to "+core.PubKeyHashToAddress(out.PubKeyHash)),
		),
	))
}

func (u *UTXOTab) update(snap *state.Snapshot) {
	u.snap = snap
	u.filter.SetOptions(append([]string{allOutputs}, snap.Addresses()...))
	u.load()
}

func (u *UTXOTab) load() {
	bc := u.state.Blockchain()
	if bc == nil {
		u.outputs = nil
		u.render(0)
		return
	}

	filter := u.filter.Text
	var pubKeyHash []byte
	if filter != "" && filter != allOutputs {
		pubKeyHash = core.AddressToPubKeyHash(filter)
		if pubKeyHash == nil {
			u.outputs = nil
			u.render(0)
			return
		}
	}

	u.rows.Objects = []fyne.CanvasObject{ui.SkeletonRows(4, 64)}
	u.rows.Refresh()

	go func() {
		set := core.UTXOSet{Blockchain: bc}
		outputs, err := set.FindUTXO(pubKeyHash)
		count, countErr := set.CountTransactions()
		fyne.Do(func() {
			if u.filter.Text != filter {
				return
			}
			if err = errors.Join(err, countErr); err != nil {
				u.state.Toast("Could not read the UTXO set", err.Error(), state.LevelError)
			}
			u.outputs = outputs
			u.render(count)
		})
	}()
}

func (u *UTXOTab) render(count int) {
	total, locked := 0, 0
	for _, out := range u.outputs {
		total += out.Value
		if out.Pending {
			locked += out.Value
		}
	}

	u.entries.Set(ui.Number(int64(count)), "keys in the chainstate bucket")
	u.shown.Set(ui.Number(int64(len(u.outputs))), "matching the filter")
	u.value.Set(ui.Number(int64(total)), "coins in these outputs")
	u.pending.Set(ui.Number(int64(locked)), "spent by pending transactions")

	u.rows.Objects = nil
	for i, out := range u.outputs {
		if i == maxRows {
			u.rows.Add(ui.Muted(fmt.Sprintf("%d more outputs, narrow the filter to see them", len(u.outputs)-maxRows)))
			break
		}
		u.rows.Add(u.row(out))
	}
	u.rows.Refresh()
}

func (u *UTXOTab) onReindex() {
	go func() {
		count, err := u.state.Reindex()
		fyne.Do(func() {
			if err != nil {
				u.state.Toast("Reindex failed", err.Error(), state.LevelError)
				return
			}
			u.state.Toast("UTXO reindexed", fmt.Sprintf("Reindex complete, %d transactions indexed", count), state.LevelSuccess)
		})
		u.state.Refresh()
	}()
}
