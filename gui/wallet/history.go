package wallet

import (
	"Blockchain/core"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strconv"
	"strings"
	"time"
)

func (w *WalletUI) showTransactionHistory(address string) {
	if !core.ValidateAddress(address) {
		dialog.ShowError(fmt.Errorf("Invalid address"), w.window)
		return
	}

	bc, err := core.NewBlockchain()
	if err != nil {
		dialog.ShowError(err, w.window)
		return
	}
	defer bc.Db.Close()

	pubKeyHash := core.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	headerText := canvas.NewText(fmt.Sprintf("Transaction history: %s", address), color.NRGBA{R: 0, G: 102, B: 204, A: 255})
	headerText.TextSize = 18
	headerText.TextStyle.Bold = true
	headerText.Alignment = fyne.TextAlignCenter
	header := container.NewHBox(
		layout.NewSpacer(),
		headerText,
		layout.NewSpacer(),
	)

	dataColor := color.NRGBA{R: 0, G: 102, B: 204, A: 255}
	receivedColor := color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	sentColor := color.NRGBA{R: 200, G: 0, B: 0, A: 255}

	createRow := func(label, value string, valueColor color.Color) *fyne.Container {
		return container.NewHBox(
			widget.NewLabel(label),
			canvas.NewText(value, valueColor),
		)
	}

	txAccordion := widget.NewAccordion()
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			for outIdx, out := range tx.Vout {
				if out.IsLockedWithKey(pubKeyHash) {
					itemTitle := fmt.Sprintf("Received %d (TX: %x...)", out.Value, tx.ID[:16])
					if outIdx == 0 {
						itemTitle += " [Main output]"
					} else {
						itemTitle += " [Change]"
					}

					item := widget.NewAccordionItem(
						itemTitle,
						container.NewVBox(
							createRow("Transaction ID:", fmt.Sprintf("%x", tx.ID), dataColor),
							createRow("Amount:", fmt.Sprintf("%d", out.Value), receivedColor),
							createRow("Type:", func() string {
								if outIdx == 0 {
									return "Main output"
								}
								return "Change output"
							}(), dataColor),
							createRow("Block:", fmt.Sprintf("%x...", block.Hash), dataColor),
							createRow("Date:", time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"), dataColor),
						),
					)
					txAccordion.Append(item)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					prevTx, err := bc.FindTransaction(in.Txid)
					if err != nil {
						continue
					}
					prevOut := prevTx.Vout[in.Vout]
					if prevOut.IsLockedWithKey(pubKeyHash) {
						item := widget.NewAccordionItem(
							fmt.Sprintf("Sent %d (TX: %x...)", prevOut.Value, tx.ID),
							container.NewVBox(
								createRow("Transaction ID:", fmt.Sprintf("%x", tx.ID), dataColor),
								createRow("Amount:", fmt.Sprintf("%d", prevOut.Value), sentColor),
								createRow("Source output:", func() string {
									if in.Vout == 0 {
										return "Main output"
									}
									return "Change output"
								}(), dataColor),
								createRow("From TX:", fmt.Sprintf("%x...", in.Txid), dataColor),
								createRow("Block:", fmt.Sprintf("%x...", block.Hash), dataColor),
								createRow("Date:", time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"), dataColor),
							),
						)
						txAccordion.Append(item)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	if len(txAccordion.Items) == 0 {
		dialog.ShowInformation("History", "No transactions found", w.window)
		return
	}

	stats := w.calculateStats(txAccordion.Items)
	statsBox := container.NewHBox(
		container.NewHBox(
			widget.NewIcon(theme.HistoryIcon()),
			canvas.NewText("Total received: ", theme.ForegroundColor()),
			canvas.NewText(fmt.Sprintf("%d \t", stats.received), color.RGBA{R: 0, G: 180, B: 0, A: 255}),
		),
		container.NewHBox(
			widget.NewIcon(theme.MailSendIcon()),
			canvas.NewText("Total sent: ", theme.ForegroundColor()),
			canvas.NewText(fmt.Sprintf("%d \t", stats.sent), color.RGBA{R: 200, G: 0, B: 0, A: 255}),
		),
		container.NewHBox(
			widget.NewIcon(theme.AccountIcon()),
			canvas.NewText("Balance: ", theme.ForegroundColor()),
			canvas.NewText(fmt.Sprintf("%d \t", stats.balance), color.RGBA{R: 0, G: 100, B: 255, A: 255}),
		),
	)

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		container.NewPadded(statsBox),
		widget.NewSeparator(),
		container.NewPadded(txAccordion),
	)

	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(800, 450))

	d := dialog.NewCustom(
		"Transaction history",
		"Close",
		scroll,
		w.window,
	)
	d.Resize(fyne.NewSize(940, 450))
	d.Show()
}

type txStats struct {
	received int
	sent     int
	balance  int
}

func (w *WalletUI) calculateStats(items []*widget.AccordionItem) txStats {
	var stats txStats
	for _, item := range items {
		if strings.Contains(item.Title, "Received") {
			parts := strings.Split(item.Title, " ")
			if len(parts) >= 2 {
				amount, _ := strconv.Atoi(parts[1])
				stats.received += amount
			}
		} else if strings.Contains(item.Title, "Sent") {
			parts := strings.Split(item.Title, " ")
			if len(parts) >= 2 {
				amount, _ := strconv.Atoi(parts[1])
				stats.sent += amount
			}
		}
	}
	stats.balance = stats.received - stats.sent
	return stats
}
