package wiki

import (
	"Blockchain/resources"
	"bytes"
	"image/png"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestSplitBlocks(t *testing.T) {
	content := "# Title\r\n\r\nIntro text.\n\n```go\nfunc a() {\n\treturn\n}\n```\n{{image:block_overview.png|A caption}}\nTail text.\n"
	chunks := splitBlocks(content)
	if len(chunks) != 4 {
		t.Fatalf("got %d chunks: %+v", len(chunks), chunks)
	}
	if chunks[0].code || !strings.Contains(chunks[0].text, "Intro text.") || strings.Contains(chunks[0].text, "\r") {
		t.Fatalf("first chunk = %+v", chunks[0])
	}
	if !chunks[1].code || chunks[1].text != "func a() {\n\treturn\n}" {
		t.Fatalf("code chunk = %q", chunks[1].text)
	}
	if chunks[2].image != "block_overview.png" || chunks[2].caption != "A caption" {
		t.Fatalf("image chunk = %+v", chunks[2])
	}
	if chunks[3].text != "Tail text." {
		t.Fatalf("tail chunk = %q", chunks[3].text)
	}

	if got := splitBlocks("```\nunterminated"); len(got) != 1 || !got[0].code {
		t.Fatalf("an unterminated fence must still produce code: %+v", got)
	}
	if got := splitBlocks("\n\n  \n"); len(got) != 0 {
		t.Fatal("blank content must produce nothing")
	}
}

func TestEveryPageAndImageExists(t *testing.T) {
	if len(pages) != 12 {
		t.Fatalf("wiki has %d chapters", len(pages))
	}
	for _, page := range pages {
		data, err := wikiFS.ReadFile("pages/" + page.FileName)
		if err != nil {
			t.Fatalf("%s: %v", page.FileName, err)
		}
		if page.Title == "" || page.Topic == "" {
			t.Errorf("%s needs a title and a topic", page.FileName)
		}
		text := string(data)
		if strings.Count(text, "```")%2 != 0 {
			t.Errorf("%s has an unclosed code fence", page.FileName)
		}
		for _, c := range splitBlocks(text) {
			if c.image == "" {
				continue
			}
			for _, name := range []string{c.image, strings.TrimSuffix(c.image, ".png") + "_dark.png"} {
				raw, err := imageFS.ReadFile("images/" + name)
				if err != nil {
					t.Errorf("%s refers to missing %s", page.FileName, name)
					continue
				}
				if _, err := png.DecodeConfig(bytes.NewReader(raw)); err != nil {
					t.Errorf("%s is not a PNG: %v", name, err)
				}
			}
			if c.caption == "" {
				t.Errorf("%s: image %s has no caption", page.FileName, c.image)
			}
		}
	}
	if _, err := imageFS.ReadFile("images/COPYING"); err != nil {
		t.Fatal("the image license notice is missing")
	}
}

func TestWikiNavigation(t *testing.T) {
	a := test.NewTempApp(t)
	a.Settings().SetTheme(&resources.Theme{Dark: true})
	t.Cleanup(func() {
		for _, page := range pages {
			page.object = nil
		}
	})

	w := New()
	win := test.NewWindow(w.Content())
	defer win.Close()
	win.Resize(fyne.NewSize(1100, 800))

	for i := range pages {
		w.open(i)
		if w.index != i || len(w.article.Objects) != 1 {
			t.Fatalf("chapter %d did not open", i)
		}
		if failed := findText(w.article.Objects[0], "Failed to load"); failed {
			t.Fatalf("chapter %d failed to render", i)
		}
	}
	current := w.index
	w.open(-1)
	w.open(len(pages))
	if w.index != current {
		t.Fatal("out of range chapters must be ignored")
	}

	if loadImage("missing.png", "") != nil {
		t.Fatal("a missing image must be skipped")
	}
	if loadImage("block_overview.png", "caption") == nil {
		t.Fatal("a known image must load")
	}
}

func findText(obj fyne.CanvasObject, needle string) bool {
	switch o := obj.(type) {
	case *fyne.Container:
		for _, child := range o.Objects {
			if findText(child, needle) {
				return true
			}
		}
	case *widget.RichText:
		return strings.Contains(o.String(), needle)
	case fyne.Widget:
		for _, child := range test.WidgetRenderer(o).Objects() {
			if findText(child, needle) {
				return true
			}
		}
	}
	return false
}
