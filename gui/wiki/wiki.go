package wiki

import (
	"Blockchain/gui/ui"
	"Blockchain/resources/icons"
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/png"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

//go:embed pages/*.md
var wikiFS embed.FS

//go:embed images/*
var imageFS embed.FS

const (
	imageWidth  = 640
	imageHeight = 420
)

type Page struct {
	FileName string
	Title    string
	Topic    string
	object   fyne.CanvasObject
}

var pages = []*Page{
	{FileName: "home.md", Title: "Introduction", Topic: "Start here"},
	{FileName: "block.md", Title: "Blocks", Topic: "Data structure"},
	{FileName: "pow.md", Title: "Proof-of-Work", Topic: "Consensus"},
	{FileName: "blockchain.md", Title: "Blockchain and storage", Topic: "Data structure"},
	{FileName: "binary_encoding.md", Title: "Binary encoding", Topic: "Data structure"},
	{FileName: "transactions.md", Title: "Transactions", Topic: "Value transfer"},
	{FileName: "wallets.md", Title: "Wallets and addresses", Topic: "Cryptography"},
	{FileName: "base58.md", Title: "Base58Check", Topic: "Cryptography"},
	{FileName: "utxo.md", Title: "UTXO set", Topic: "Value transfer"},
	{FileName: "merkle_tree.md", Title: "Merkle tree", Topic: "Data structure"},
	{FileName: "mempool.md", Title: "Mempool and mining", Topic: "Value transfer"},
	{FileName: "p2p_network.md", Title: "P2P network", Topic: "Consensus"},
}

type WikiUI struct {
	list    *widget.List
	article *fyne.Container
	index   int
	content fyne.CanvasObject
}

func New() *WikiUI {
	w := &WikiUI{index: -1}
	w.content = w.build()
	w.list.Select(0)
	return w
}

func (w *WikiUI) Content() fyne.CanvasObject {
	return w.content
}

func (w *WikiUI) build() fyne.CanvasObject {
	w.list = widget.NewList(
		func() int { return len(pages) },
		func() fyne.CanvasObject {
			return ui.Padded(4, 4, 8, 8, container.NewVBox(ui.NewText("", 14, theme.ColorNameForeground), ui.Caption("")))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			box := item.(*fyne.Container).Objects[0].(*fyne.Container)
			box.Objects[0].(*ui.Text).SetText(pages[id].Title)
			box.Objects[1].(*ui.Text).SetText(pages[id].Topic)
		},
	)
	w.list.OnSelected = w.open

	w.article = container.NewStack()

	nav := ui.NewSurface(container.NewBorder(
		ui.Padded(0, 8, 8, 0, container.NewHBox(ui.Icon(icons.ResourceWikiPng, 24), container.NewCenter(ui.Section("Chapters", "")))),
		nil, nil, nil,
		w.list,
	))

	article := ui.NewSurface(w.article)
	article.Padding = 24

	split := container.NewHSplit(nav, article)
	split.Offset = 0.24
	return split
}

func (w *WikiUI) open(index int) {
	if index == w.index || index < 0 || index >= len(pages) {
		return
	}
	w.index = index

	page := pages[index]
	if page.object == nil {
		page.object = w.render(index)
	}

	w.article.Objects = []fyne.CanvasObject{page.object}
	w.article.Refresh()
}

func (w *WikiUI) render(index int) fyne.CanvasObject {
	page := pages[index]
	data, err := wikiFS.ReadFile("pages/" + page.FileName)
	if err != nil {
		return ui.Muted("Failed to load page: " + err.Error())
	}

	body := container.NewVBox(
		ui.Caption(page.Topic),
	)
	for _, block := range splitBlocks(string(data)) {
		switch {
		case block.image != "":
			if img := loadImage(block.image, block.caption); img != nil {
				body.Add(ui.Padded(8, 8, 0, 0, img))
			}
		case block.code:
			body.Add(codeBlock(block.text))
		default:
			text := widget.NewRichTextFromMarkdown(block.text)
			text.Wrapping = fyne.TextWrapWord
			body.Add(text)
		}
	}

	if page.FileName == "home.md" {
		body.Add(w.chapterGrid())
	}

	prev := widget.NewButtonWithIcon("Previous", theme.NavigateBackIcon(), func() { w.list.Select(index - 1) })
	next := widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() { w.list.Select(index + 1) })
	next.IconPlacement = widget.ButtonIconTrailingText
	if index == 0 {
		prev.Disable()
	}
	if index == len(pages)-1 {
		next.Disable()
	}

	footer := container.NewBorder(nil, nil, prev, next,
		container.NewCenter(ui.Caption(fmt.Sprintf("%d of %d", index+1, len(pages)))))

	scroll := ui.NewNestedScroll(ui.Padded(0, 0, 8, 16, body))
	return container.NewBorder(nil, ui.Padded(10, 0, 0, 0, footer), nil, nil, scroll)
}

func (w *WikiUI) chapterGrid() fyne.CanvasObject {
	grid := container.NewGridWithColumns(3)
	for i, page := range pages[1:] {
		index := i + 1
		grid.Add(ui.NewTapSurface(container.NewVBox(
			ui.NewText(page.Title, 14, theme.ColorNameForeground),
			ui.Caption(page.Topic),
		), func() { w.list.Select(index) }))
	}
	return grid
}

type chunk struct {
	text    string
	image   string
	caption string
	code    bool
}

func splitBlocks(content string) []chunk {
	var chunks []chunk
	var text []string
	inCode := false

	flush := func(code bool) {
		joined := strings.Join(text, "\n")
		if strings.TrimSpace(joined) != "" {
			chunks = append(chunks, chunk{text: strings.TrimRight(joined, "\n"), code: code})
		}
		text = text[:0]
	}

	for _, line := range strings.Split(strings.ReplaceAll(content, "\r", ""), "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			flush(inCode)
			inCode = !inCode
			continue
		}
		if inCode {
			text = append(text, line)
			continue
		}

		if strings.HasPrefix(trimmed, "{{image:") && strings.HasSuffix(trimmed, "}}") {
			flush(false)
			name, caption, _ := strings.Cut(trimmed[len("{{image:"):len(trimmed)-2], "|")
			chunks = append(chunks, chunk{image: strings.TrimSpace(name), caption: strings.TrimSpace(caption)})
			continue
		}

		text = append(text, line)
	}
	flush(inCode)

	return chunks
}

func codeBlock(code string) fyne.CanvasObject {
	label := widget.NewLabel(strings.ReplaceAll(code, "\t", "    "))
	label.TextStyle = fyne.TextStyle{Monospace: true}
	label.Wrapping = fyne.TextWrapBreak

	return ui.Padded(4, 4, 0, 0, ui.NewRaised(label))
}

func loadImage(name, caption string) fyne.CanvasObject {
	light, err := imageFS.ReadFile("images/" + name)
	if err != nil {
		return nil
	}

	dark := light
	darkName := strings.TrimSuffix(name, ".png") + "_dark.png"
	if data, err := imageFS.ReadFile("images/" + darkName); err == nil {
		dark = data
	}

	width, height := float32(imageWidth), float32(imageWidth*0.6)
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(light)); err == nil && cfg.Width > 0 {
		width = fyne.Min(imageWidth, float32(cfg.Width)/2.5)
		height = width * float32(cfg.Height) / float32(cfg.Width)
		if height > imageHeight {
			width = width * imageHeight / height
			height = imageHeight
		}
	}

	img := newDiagram(fyne.NewStaticResource(name, light), fyne.NewStaticResource(darkName, dark), fyne.NewSize(width, height))
	content := container.NewVBox(container.NewCenter(img))
	if caption != "" {
		content.Add(container.NewCenter(ui.Caption(caption)))
	}

	return content
}

type diagram struct {
	widget.BaseWidget

	light, dark fyne.Resource
	size        fyne.Size
}

func newDiagram(light, dark fyne.Resource, size fyne.Size) *diagram {
	d := &diagram{light: light, dark: dark, size: size}
	d.ExtendBaseWidget(d)
	return d
}

func (d *diagram) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromResource(d.light)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(d.size)
	r := &diagramRenderer{d: d, img: img}
	r.Refresh()
	return r
}

type diagramRenderer struct {
	d   *diagram
	img *canvas.Image
}

func (r *diagramRenderer) Destroy() {}

func (r *diagramRenderer) Layout(size fyne.Size) {
	r.img.Resize(size)
}

func (r *diagramRenderer) MinSize() fyne.Size {
	return r.d.size
}

func (r *diagramRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.img}
}

func (r *diagramRenderer) Refresh() {
	res := r.d.light
	if isDark() {
		res = r.d.dark
	}
	if r.img.Resource != res {
		r.img.Resource = res
		r.img.Refresh()
	}
}

func isDark() bool {
	red, green, blue, _ := theme.Color(theme.ColorNameBackground).RGBA()
	return red+green+blue < 3*0x8000
}
