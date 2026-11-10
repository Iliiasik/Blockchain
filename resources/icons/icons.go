package icons

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed *.png
var files embed.FS

func load(name string) fyne.Resource {
	data, err := files.ReadFile(name)
	if err != nil {
		panic("missing embedded icon " + name)
	}

	return fyne.NewStaticResource(name, data)
}

var (
	ResourceAboutPng        = load("about.png")
	ResourceBlockchainPng   = load("blockchain.png")
	ResourceCoinPng         = load("coin.png")
	ResourceIconPng         = load("icon.png")
	ResourceInfoPng         = load("info.png")
	ResourceMiningPng       = load("mining.png")
	ResourceP2pNetworkPng   = load("p2p-network.png")
	ResourceTransactionsPng = load("transactions.png")
	ResourceUtxoPng         = load("utxo.png")
	ResourceWalletsPng      = load("wallets.png")
	ResourceWikiPng         = load("wiki.png")
)
