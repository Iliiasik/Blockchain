package p2p_demo

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (p *P2PDemoUI) showNotification(message string) {
	fyne.Do(func() {
		dialog.ShowInformation(
			"P2P simulation",
			message,
			p.window,
		)
	})
}
