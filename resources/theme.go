package resources

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

//go:embed OfficeCodePro-Light.otf
var officeCodePro []byte

var officeCode fyne.Resource

func init() {
	officeCode = fyne.NewStaticResource("OfficeCodePro-Light.otf", officeCodePro)
}

type CustomTheme struct{}

func (t *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (t *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return officeCode
}

func (t *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	base := theme.DefaultTheme().Size(name)
	switch name {
	case theme.SizeNameText, theme.SizeNameCaptionText, theme.SizeNameHeadingText:
		return base + 4
	default:
		return base
	}
}
