package resources

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

//go:embed font/Technor-Regular.otf
var technorRegular []byte

var technor fyne.Resource

func init() {
	technor = fyne.NewStaticResource("Technor-Regular.otf", technorRegular)
}

type DarkTheme struct{}

func (t *DarkTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (t *DarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return technor
}

func (t *DarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *DarkTheme) Size(name fyne.ThemeSizeName) float32 {
	base := theme.DefaultTheme().Size(name)
	switch name {
	case theme.SizeNameText, theme.SizeNameCaptionText:
		return base + 2
	case theme.SizeNameHeadingText:
		return base + 4
	default:
		return base
	}
}

type LightTheme struct{}

func (t *LightTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {

	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (t *LightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return technor
}

func (t *LightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *LightTheme) Size(name fyne.ThemeSizeName) float32 {
	return (&DarkTheme{}).Size(name)
}
