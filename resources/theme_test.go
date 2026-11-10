package resources

import (
	"bytes"
	"image/color"
	"log"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestPalettesDefineEveryCustomColor(t *testing.T) {
	names := []fyne.ThemeColorName{ColorNameSurface, ColorNameRaised, ColorNameBorder, ColorNameSecondary, ColorNameMuted, ColorNameAccent, theme.ColorNamePrimary, theme.ColorNameBackground, theme.ColorNameForeground}
	dark, light := &Theme{Dark: true}, &Theme{}
	for _, name := range names {
		d, l := dark.Color(name, theme.VariantDark), light.Color(name, theme.VariantLight)
		if d == nil || l == nil {
			t.Fatalf("%s is missing", name)
		}
	}
	if colorsEqual(dark.Color(theme.ColorNameBackground, 0), light.Color(theme.ColorNameBackground, 0)) {
		t.Fatal("dark and light backgrounds must differ")
	}
	if dark.Color(theme.ColorNameHover, theme.VariantDark) == nil {
		t.Fatal("unknown names must fall back to the default theme")
	}
}

func colorsEqual(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func TestFonts(t *testing.T) {
	th := &Theme{}
	if th.Font(fyne.TextStyle{}) != SoraRegular || th.Font(fyne.TextStyle{Bold: true}) != SoraSemiBold {
		t.Fatal("text must use Sora")
	}
	if th.Font(fyne.TextStyle{Monospace: true}) == SoraRegular || th.Font(fyne.TextStyle{Monospace: true}).Name() != theme.DefaultTheme().Font(fyne.TextStyle{Monospace: true}).Name() {
		t.Fatal("code must stay in the default monospaced font")
	}
	for _, font := range []fyne.Resource{SoraRegular, SoraMedium, SoraSemiBold} {
		if !bytes.HasPrefix(font.Content(), []byte{0, 1, 0, 0}) {
			t.Fatalf("%s is not a TrueType font", font.Name())
		}
	}
}

func TestLineHeightIsComfortable(t *testing.T) {
	test.NewTempApp(t).Settings().SetTheme(&Theme{})
	for _, style := range []fyne.TextStyle{{}, {Bold: true}, {Monospace: true}, {Monospace: true, Bold: true}} {
		size := fyne.MeasureText("Ag", 14, style)
		if ratio := size.Height / 14; ratio < 1.45 || ratio > 1.55 {
			t.Errorf("%+v: line height is %.2f of the font size, want about 1.5", style, ratio)
		}
	}
}

func TestLineHeightPatchKeepsBrokenFontsIntact(t *testing.T) {
	broken := []byte("not a font")
	if got := withLineHeight("x.ttf", broken).Content(); string(got) != string(broken) {
		t.Fatal("unknown data must pass through untouched")
	}
	original := append([]byte(nil), soraRegular...)
	withLineHeight("copy.ttf", soraRegular)
	if !bytes.Equal(original, soraRegular) {
		t.Fatal("patching must not modify the embedded font")
	}
}

func TestSizes(t *testing.T) {
	th := &Theme{}
	if th.Size(theme.SizeNameText) != 14 || th.Size(theme.SizeNamePadding) != 4 {
		t.Fatal("base sizes changed")
	}
	if th.Size(theme.SizeNameLineSpacing) < 8 {
		t.Fatal("paragraph spacing is too tight")
	}
	if th.Size(theme.SizeNameInlineIcon) != theme.DefaultTheme().Size(theme.SizeNameInlineIcon) {
		t.Fatal("unknown sizes must fall back to the default theme")
	}
	if th.Icon(theme.IconNameHome) == nil {
		t.Fatal("icons must come from the default theme")
	}
}

func TestUseEnglishOverridesSystemLanguage(t *testing.T) {
	t.Setenv("LANGUAGE", "ru_RU")
	t.Setenv("LC_ALL", "ru_RU.UTF-8")
	t.Setenv("LC_MESSAGES", "ru_RU.UTF-8")
	t.Setenv("LANG", "ru_RU.UTF-8")

	if err := lang.AddTranslationsForLocale([]byte("{}"), fyne.Locale("xx")); err != nil {
		t.Fatal(err)
	}
	if lang.L("Cancel") == "Cancel" {
		t.Skip("the system did not switch to Russian, nothing to override")
	}

	UseEnglish()

	var logged bytes.Buffer
	log.SetOutput(&logged)
	defer log.SetOutput(os.Stderr)

	for _, key := range []string{"(Parent)", "(Select one)", "Advanced", "Cancel", "Confirm", "Copy", "Create Folder", "Cut", "Enter filename", "Error", "File", "Folder", "New Folder", "No", "OK", "Open", "Paste", "Quit", "Redo", "Save", "Select all", "Show Hidden Files", "Undo", "Yes"} {
		if got := lang.L(key); got != key {
			t.Errorf("%s is shown as %q", key, got)
		}
	}
	for key, want := range map[string]string{"file.name": "Name", "file.parent": "Parent", "example": "Example", "label.description": "Description", "monday": "Monday", "sunday.short": "Sun"} {
		if got := lang.X(key, "fallback"); got != want {
			t.Errorf("%s is shown as %q, want %q", key, got, want)
		}
	}
	if logged.Len() > 0 {
		t.Errorf("Fyne reported missing translations:\n%s", logged.String())
	}
}
