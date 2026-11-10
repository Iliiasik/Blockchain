package ui

import (
	"Blockchain/core"
	"Blockchain/resources"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func Short(data []byte) string {
	return ShortString(hex.EncodeToString(data), 10, 6)
}

func ShortString(s string, head, tail int) string {
	if len(s) <= head+tail+1 {
		return s
	}
	return s[:head] + "..." + s[len(s)-tail:]
}

func ShortAddress(address string) string {
	return ShortString(address, 8, 6)
}

func FormatTime(ts int64) string {
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

func Ago(ts int64) string {
	d := time.Since(time.Unix(ts, 0))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d d ago", int(d.Hours()/24))
}

func Number(n int64) string {
	if n < 0 {
		return "-" + Number(-n)
	}
	digits := strconv.FormatInt(n, 10)
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i := 0; i < len(digits); i++ {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ' ')
		}
		out = append(out, digits[i])
	}
	return string(out)
}

func Target(bits int) string {
	target := new(big.Int).Lsh(big.NewInt(1), uint(256-bits))
	return fmt.Sprintf("%064x", target)
}

func Attempts(bits int) string {
	return Number(int64(1) << uint(bits))
}

func BlockView(block *core.Block, height int, notify func(string)) fyne.CanvasObject {
	valid := core.NewProofOfWork(block).Validate()

	status := NewBadge("PoW valid", theme.ColorNameSuccess)
	if !valid {
		status.Set("PoW invalid", theme.ColorNameError)
	}
	badges := container.NewHBox(status)
	if len(block.PrevBlockHash) == 0 {
		badges.Add(NewBadge("Genesis", resources.ColorNameMuted))
	}

	prev := "none (genesis block)"
	if len(block.PrevBlockHash) > 0 {
		prev = hex.EncodeToString(block.PrevBlockHash)
	}

	header := container.NewVBox(
		container.NewBorder(nil, nil, Title(fmt.Sprintf("Block #%d", height), 24), container.NewCenter(badges)),
		Copyable(hex.EncodeToString(block.Hash), notify),
	)

	fields := KV(
		Key("Timestamp"), Value(FormatTime(block.Timestamp)),
		Key("Previous"), Copyable(prev, notify),
		Key("Merkle root"), Copyable(hex.EncodeToString(block.HashTransactions()), notify),
		Key("Nonce"), Value(Number(int64(block.Nonce))),
		Key("Difficulty"), Value(fmt.Sprintf("%d bits, about %s attempts", block.Bits, Attempts(block.Bits))),
		Key("Target"), MonoValue(Target(block.Bits)),
		Key("Size"), Value(fmt.Sprintf("%s bytes", Number(int64(len(block.Serialize()))))),
	)

	txs := container.NewVBox()
	for i, tx := range block.Transactions {
		txs.Add(TransactionView(tx, i, notify))
	}

	return container.NewVBox(
		header,
		Divider(),
		fields,
		Padded(16, 4, 0, 0, Section(fmt.Sprintf("Transactions (%d)", len(block.Transactions)), "")),
		txs,
	)
}

func TransactionView(tx *core.Transaction, index int, notify func(string)) fyne.CanvasObject {
	kind := NewBadge("Transfer", resources.ColorNameAccent)
	if tx.IsCoinbase() {
		kind.Set("Coinbase", theme.ColorNameSuccess)
	}

	head := container.NewBorder(nil, nil,
		container.NewHBox(container.NewCenter(NewText(fmt.Sprintf("#%d", index), 13, resources.ColorNameMuted)), container.NewCenter(kind)),
		container.NewCenter(NewText(fmt.Sprintf("Fee %d", tx.Fee), 13, resources.ColorNameMuted)),
		Copyable(hex.EncodeToString(tx.ID), notify),
	)

	inputs := container.NewVBox(Caption("Inputs"))
	for _, in := range tx.Vin {
		if tx.IsCoinbase() {
			inputs.Add(Secondary("New coins, no previous output"))
			inputs.Add(Paragraph(fmt.Sprintf("\"%s\"", string(in.PubKey))))
			continue
		}
		inputs.Add(container.NewVBox(
			MonoValue(fmt.Sprintf("%s : %d", Short(in.Txid), in.Vout)),
			NewText("From "+core.PubKeyToAddress(in.PubKey), 12, resources.ColorNameMuted),
		))
	}

	outputs := container.NewVBox(Caption("Outputs"))
	for i, out := range tx.Vout {
		outputs.Add(container.NewVBox(
			container.NewHBox(
				NewText(fmt.Sprintf("#%d", i), 13, resources.ColorNameMuted),
				Strong(Number(int64(out.Value)), 14),
			),
			NewText("To "+core.PubKeyHashToAddress(out.PubKeyHash), 12, resources.ColorNameMuted),
		))
	}

	return NewRaised(container.NewVBox(
		head,
		Divider(),
		container.NewGridWithColumns(2, inputs, outputs),
	))
}
