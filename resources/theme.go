package resources

import (
	_ "embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed font/Sora-Regular.ttf
var soraRegular []byte

//go:embed font/Sora-Medium.ttf
var soraMedium []byte

//go:embed font/Sora-SemiBold.ttf
var soraSemiBold []byte

var (
	SoraRegular  = withLineHeight("Sora-Regular.ttf", soraRegular)
	SoraMedium   = withLineHeight("Sora-Medium.ttf", soraMedium)
	SoraSemiBold = withLineHeight("Sora-SemiBold.ttf", soraSemiBold)
)

const (
	ColorNameSurface   fyne.ThemeColorName = "surface"
	ColorNameRaised    fyne.ThemeColorName = "raised"
	ColorNameBorder    fyne.ThemeColorName = "border"
	ColorNameSecondary fyne.ThemeColorName = "secondary"
	ColorNameMuted     fyne.ThemeColorName = "muted"
	ColorNameAccent    fyne.ThemeColorName = "accent"
)

type palette map[fyne.ThemeColorName]color.Color

func hex(rgb uint32, alpha uint8) color.NRGBA {
	return color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: alpha}
}

var darkPalette = palette{
	theme.ColorNameBackground:          hex(0x141415, 0xFF),
	theme.ColorNameHeaderBackground:    hex(0x0F0F10, 0xFF),
	theme.ColorNameMenuBackground:      hex(0x19191B, 0xFF),
	theme.ColorNameOverlayBackground:   hex(0x19191B, 0xFF),
	theme.ColorNameInputBackground:     hex(0x232325, 0xFF),
	theme.ColorNameInputBorder:         hex(0xEDEDED, 0x1A),
	theme.ColorNameButton:              hex(0x232325, 0xFF),
	theme.ColorNameDisabledButton:      hex(0x1C1C1E, 0xFF),
	theme.ColorNameHover:               hex(0xEDEDED, 0x0F),
	theme.ColorNamePressed:             hex(0xEDEDED, 0x1A),
	theme.ColorNameForeground:          hex(0xEDEDED, 0xFF),
	theme.ColorNameDisabled:            hex(0x5C5C61, 0xFF),
	theme.ColorNamePlaceHolder:         hex(0x85858B, 0xFF),
	theme.ColorNamePrimary:             hex(0x2F6FEB, 0xFF),
	theme.ColorNameForegroundOnPrimary: hex(0xFFFFFF, 0xFF),
	theme.ColorNameFocus:               hex(0x2F6FEB, 0x99),
	theme.ColorNameSelection:           hex(0x2F6FEB, 0x40),
	theme.ColorNameHyperlink:           hex(0x7AA7FF, 0xFF),
	theme.ColorNameSeparator:           hex(0xEDEDED, 0x14),
	theme.ColorNameShadow:              hex(0x000000, 0x66),
	theme.ColorNameScrollBar:           hex(0xEDEDED, 0x29),
	theme.ColorNameScrollBarBackground: hex(0x000000, 0x00),
	theme.ColorNameSuccess:             hex(0x4CC38A, 0xFF),
	theme.ColorNameForegroundOnSuccess: hex(0x0F0F10, 0xFF),
	theme.ColorNameWarning:             hex(0xE0A43A, 0xFF),
	theme.ColorNameForegroundOnWarning: hex(0x0F0F10, 0xFF),
	theme.ColorNameError:               hex(0xE5484D, 0xFF),
	theme.ColorNameForegroundOnError:   hex(0xFFFFFF, 0xFF),
	ColorNameSurface:                   hex(0x19191B, 0xFF),
	ColorNameRaised:                    hex(0x232325, 0xFF),
	ColorNameBorder:                    hex(0xEDEDED, 0x14),
	ColorNameSecondary:                 hex(0xA1A1A6, 0xFF),
	ColorNameMuted:                     hex(0x85858B, 0xFF),
	ColorNameAccent:                    hex(0x7AA7FF, 0xFF),
}

var lightPalette = palette{
	theme.ColorNameBackground:          hex(0xFAFAFA, 0xFF),
	theme.ColorNameHeaderBackground:    hex(0xF4F4F5, 0xFF),
	theme.ColorNameMenuBackground:      hex(0xFFFFFF, 0xFF),
	theme.ColorNameOverlayBackground:   hex(0xFFFFFF, 0xFF),
	theme.ColorNameInputBackground:     hex(0xF4F4F5, 0xFF),
	theme.ColorNameInputBorder:         hex(0x171717, 0x1F),
	theme.ColorNameButton:              hex(0xF4F4F5, 0xFF),
	theme.ColorNameDisabledButton:      hex(0xEFEFF0, 0xFF),
	theme.ColorNameHover:               hex(0x171717, 0x0D),
	theme.ColorNamePressed:             hex(0x171717, 0x17),
	theme.ColorNameForeground:          hex(0x171717, 0xFF),
	theme.ColorNameDisabled:            hex(0xA3A3A3, 0xFF),
	theme.ColorNamePlaceHolder:         hex(0x737373, 0xFF),
	theme.ColorNamePrimary:             hex(0x2563EB, 0xFF),
	theme.ColorNameForegroundOnPrimary: hex(0xFFFFFF, 0xFF),
	theme.ColorNameFocus:               hex(0x2563EB, 0x80),
	theme.ColorNameSelection:           hex(0x2563EB, 0x2E),
	theme.ColorNameHyperlink:           hex(0x1D4ED8, 0xFF),
	theme.ColorNameSeparator:           hex(0x171717, 0x14),
	theme.ColorNameShadow:              hex(0x000000, 0x14),
	theme.ColorNameScrollBar:           hex(0x171717, 0x29),
	theme.ColorNameScrollBarBackground: hex(0x000000, 0x00),
	theme.ColorNameSuccess:             hex(0x16A34A, 0xFF),
	theme.ColorNameForegroundOnSuccess: hex(0xFFFFFF, 0xFF),
	theme.ColorNameWarning:             hex(0xB7791F, 0xFF),
	theme.ColorNameForegroundOnWarning: hex(0xFFFFFF, 0xFF),
	theme.ColorNameError:               hex(0xDC2626, 0xFF),
	theme.ColorNameForegroundOnError:   hex(0xFFFFFF, 0xFF),
	ColorNameSurface:                   hex(0xFFFFFF, 0xFF),
	ColorNameRaised:                    hex(0xF4F4F5, 0xFF),
	ColorNameBorder:                    hex(0x171717, 0x14),
	ColorNameSecondary:                 hex(0x525252, 0xFF),
	ColorNameMuted:                     hex(0x737373, 0xFF),
	ColorNameAccent:                    hex(0x1D4ED8, 0xFF),
}

type Theme struct {
	Dark bool
}

func (t *Theme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	colors := lightPalette
	if t.Dark {
		colors = darkPalette
	}
	if c, ok := colors[name]; ok {
		return c
	}

	variant := theme.VariantLight
	if t.Dark {
		variant = theme.VariantDark
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (t *Theme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Symbol {
		return theme.DefaultTheme().Font(style)
	}
	if style.Monospace {
		return monospace(style.Bold)
	}
	if style.Bold {
		return SoraSemiBold
	}

	return SoraRegular
}

func (t *Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *Theme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameHeadingText:
		return 24
	case theme.SizeNameSubHeadingText:
		return 18
	case theme.SizeNameLineSpacing:
		return 10
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameInputRadius, theme.SizeNameButtonRadius, theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameCardRadius, theme.SizeNameMenuRadius:
		return 12
	case theme.SizeNameDialogRadius, theme.SizeNamePopupRadius:
		return 16
	case theme.SizeNameScrollBar:
		return 8
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameScrollBarRadius:
		return 4
	case theme.SizeNameSeparatorThickness, theme.SizeNameInputBorder:
		return 1
	}

	return theme.DefaultTheme().Size(name)
}
