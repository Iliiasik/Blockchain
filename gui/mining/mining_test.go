package mining

import (
	"Blockchain/gui/testkit"
	"testing"
	"time"
)

func TestMiningScreen(t *testing.T) {
	env := testkit.New(t, 2, true)
	m := New(env.State)
	env.Show(m.Content())
	testkit.Sync(t, env.State)

	if len(m.miner.Options) != 2 {
		t.Fatalf("miner options = %v", m.miner.Options)
	}
	m.Focus(env.Wallets[1])
	if m.miner.Selected != env.Wallets[1] || m.start.Disabled() {
		t.Fatal("start must be enabled once a miner is chosen")
	}
	if m.abort.Disabled() == false {
		t.Fatal("abort must be disabled while idle")
	}
	if m.reward.Text == "" {
		t.Fatal("the reward summary is empty")
	}
}

func TestFormatting(t *testing.T) {
	if formatDuration(1500*time.Millisecond) == "" || formatRate(123456) == "" {
		t.Fatal("formatters returned nothing")
	}
}
