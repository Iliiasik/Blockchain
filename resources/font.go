package resources

import (
	"encoding/binary"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const (
	ascentShare  = 1.09
	descentShare = 0.41
)

var monoFonts = sync.OnceValue(func() [2]fyne.Resource {
	regular := theme.DefaultTheme().Font(fyne.TextStyle{Monospace: true})
	bold := theme.DefaultTheme().Font(fyne.TextStyle{Monospace: true, Bold: true})
	return [2]fyne.Resource{
		withLineHeight(regular.Name(), regular.Content()),
		withLineHeight(bold.Name(), bold.Content()),
	}
})

func monospace(bold bool) fyne.Resource {
	if bold {
		return monoFonts()[1]
	}
	return monoFonts()[0]
}

func withLineHeight(name string, data []byte) fyne.Resource {
	font := append([]byte(nil), data...)
	tables := fontTables(font)
	head, hasHead := tables["head"]
	hhea, hasHhea := tables["hhea"]
	if !hasHead || !hasHhea || head+20 > len(font) || hhea+10 > len(font) {
		return fyne.NewStaticResource(name, data)
	}

	unitsPerEm := float64(binary.BigEndian.Uint16(font[head+18:]))
	ascent := int16(math.Round(unitsPerEm * ascentShare))
	descent := int16(-math.Round(unitsPerEm * descentShare))

	putInt16(font[hhea+4:], ascent)
	putInt16(font[hhea+6:], descent)
	putInt16(font[hhea+8:], 0)

	if os2, ok := tables["OS/2"]; ok && os2+78 <= len(font) {
		putInt16(font[os2+68:], ascent)
		putInt16(font[os2+70:], descent)
		putInt16(font[os2+72:], 0)
		binary.BigEndian.PutUint16(font[os2+74:], max(binary.BigEndian.Uint16(font[os2+74:]), uint16(ascent)))
		binary.BigEndian.PutUint16(font[os2+76:], max(binary.BigEndian.Uint16(font[os2+76:]), uint16(-descent)))
	}

	return fyne.NewStaticResource(name, font)
}

func fontTables(font []byte) map[string]int {
	tables := map[string]int{}
	if len(font) < 12 {
		return tables
	}
	count := int(binary.BigEndian.Uint16(font[4:]))
	for i := 0; i < count; i++ {
		record := 12 + 16*i
		if record+16 > len(font) {
			break
		}
		tables[string(font[record:record+4])] = int(binary.BigEndian.Uint32(font[record+8:]))
	}
	return tables
}

func putInt16(dst []byte, value int16) {
	binary.BigEndian.PutUint16(dst, uint16(value))
}
