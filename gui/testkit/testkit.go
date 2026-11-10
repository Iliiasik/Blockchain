package testkit

import (
	"Blockchain/gui/state"
	"Blockchain/resources"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

type Env struct {
	State   *state.AppState
	Window  fyne.Window
	Wallets []string
}

func New(t *testing.T, wallets int, withChain bool) *Env {
	t.Helper()
	test.NewTempApp(t).Settings().SetTheme(&resources.Theme{Dark: true})
	t.Chdir(t.TempDir())

	st := state.NewAppState()
	st.TargetBits = 8
	env := &Env{State: st, Window: test.NewWindow(nil)}
	env.Window.Resize(fyne.NewSize(1280, 860))

	for range wallets {
		address, err := st.CreateWallet()
		if err != nil {
			t.Fatal(err)
		}
		env.Wallets = append(env.Wallets, address)
	}
	if withChain {
		if err := st.CreateBlockchain(env.Wallets[0]); err != nil {
			t.Fatal(err)
		}
	}

	t.Cleanup(func() {
		Settle(t, st)
		env.Window.Close()
		if err := st.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return env
}

func (e *Env) Show(content fyne.CanvasObject) {
	e.Window.SetContent(content)
}

func Settle(t *testing.T, st *state.AppState) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	quiet := 0
	for quiet < 3 {
		if time.Now().After(deadline) {
			t.Fatal("background work did not settle")
		}
		if st.Busy() {
			quiet = 0
		} else {
			quiet++
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func Sync(t *testing.T, st *state.AppState) *state.Snapshot {
	t.Helper()
	Settle(t, st)
	st.Refresh()
	Settle(t, st)
	return st.Snapshot()
}
