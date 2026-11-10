package about

import (
	"Blockchain/gui/testkit"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func TestOverviewProgress(t *testing.T) {
	env := testkit.New(t, 1, true)
	o := New(env.State)
	env.Show(o.Content())
	testkit.Sync(t, env.State)

	if o.blocks.Value.Text != "1" || o.supply.Value.Text != "10" || o.difficulty.Value.Text != "8 bits" {
		t.Fatalf("stats = %s %s %s", o.blocks.Value.Text, o.supply.Value.Text, o.difficulty.Value.Text)
	}
	want := []fyne.ThemeColorName{theme.ColorNameSuccess, theme.ColorNameSuccess, theme.ColorNamePrimary, theme.ColorNameDisabled, theme.ColorNameDisabled}
	for i, obj := range o.objectives {
		if obj.status.ColorName != want[i] {
			t.Errorf("objective %d is %s, want %s", i+1, obj.status.ColorName, want[i])
		}
	}
}
