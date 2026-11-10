package icons

import (
	"bytes"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
)

func TestIconsAreValidImages(t *testing.T) {
	all := []fyne.Resource{
		ResourceAboutPng, ResourceBlockchainPng, ResourceCoinPng, ResourceIconPng, ResourceInfoPng, ResourceMiningPng,
		ResourceP2pNetworkPng, ResourceTransactionsPng, ResourceUtxoPng, ResourceWalletsPng, ResourceWikiPng,
	}
	for _, res := range all {
		cfg, err := png.DecodeConfig(bytes.NewReader(res.Content()))
		if err != nil {
			t.Fatalf("%s: %v", res.Name(), err)
		}
		if cfg.Width < 32 || cfg.Height < 32 {
			t.Errorf("%s is only %dx%d", res.Name(), cfg.Width, cfg.Height)
		}
	}
}

func TestMissingIconPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("loading a missing icon must panic at startup")
		}
	}()
	load("missing.png")
}
