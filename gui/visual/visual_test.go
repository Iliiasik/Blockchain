package visual

import (
	"Blockchain/gui/state"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestGuideScript(t *testing.T) {
	themedApp(t)
	g := &guide{scene: NewScene()}
	w := test.NewWindow(g.scene)
	defer w.Close()
	w.Resize(fyne.NewSize(860, 430))
	g.scene.SetModel(demoModel(false), false)

	steps := g.script()
	if len(steps) != 8 {
		t.Fatalf("guide has %d steps", len(steps))
	}
	mining := 0
	for i, step := range steps {
		if step.title == "" || len(step.text) < 80 {
			t.Errorf("step %d needs a title and a real explanation", i+1)
		}
		for _, r := range step.focus {
			if !g.scene.known(r) {
				t.Errorf("step %d focuses on %+v, which is not in the demo", i+1, r)
			}
		}
		c := step.view(g)
		if c.dist <= 0 || math.IsNaN(c.yaw) || math.IsNaN(c.target.x) {
			t.Errorf("step %d has a broken camera: %+v", i+1, c)
		}
		if step.mining {
			mining++
		}
	}
	if mining != 1 || !steps[len(steps)-1].spin {
		t.Fatal("exactly one step mines and the last one spins")
	}
}

func TestVisualizerScreen(t *testing.T) {
	themedApp(t)
	t.Chdir(t.TempDir())
	st := state.NewAppState()
	w := test.NewWindow(nil)
	defer w.Close()

	v := New(st, w)
	w.SetContent(v.Content())
	w.Resize(fyne.NewSize(1200, 800))

	if v.mine.Disabled() == false || v.transfer.Disabled() == false {
		t.Fatal("actions must be disabled without a blockchain")
	}
	if len(v.card.Objects) != 0 {
		t.Fatal("no card without a selection")
	}

	v.model = demoModel(false)
	v.scene.SetModel(v.model, false)
	v.scene.Select(blockRef(1))
	if len(v.card.Objects) != 1 {
		t.Fatal("selecting a block must show its card")
	}
	if v.openAction(blockRef(1)) == nil || v.openAction(walletRef("bob")) == nil || v.openAction(txRef(demoPending)) == nil {
		t.Fatal("every selection kind needs an open action")
	}
	if v.openAction(ref{}) != nil || v.openAction(txRef("missing")) != nil {
		t.Fatal("nothing to open for an empty or unknown selection")
	}
	v.scene.Select(ref{})
	if len(v.card.Objects) != 0 {
		t.Fatal("clearing the selection must hide the card")
	}

	v.toggleSpin()
	if !v.scene.spin || v.spin.Text != "Stop rotating" {
		t.Fatal("auto rotate did not start")
	}
	v.toggleSpin()
	if v.scene.spin || v.spin.Text != "Auto rotate" {
		t.Fatal("auto rotate did not stop")
	}

	v.SetActive(false)
	if !v.scene.paused {
		t.Fatal("a hidden tab must pause the scene")
	}
	v.SetActive(true)
	if v.scene.paused {
		t.Fatal("showing the tab must resume the scene")
	}
}
