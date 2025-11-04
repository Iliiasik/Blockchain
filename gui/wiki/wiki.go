package wiki

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"Blockchain/resources/icons"
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

type Page struct {
	FileName string
	Title    string
	Content  string
}

type WikiUI struct {
	tab      *container.TabItem
	pages    []Page
	index    int
	title    *widget.Label
	btnPrev  *widget.Button
	btnNext  *widget.Button
	scroll   *container.Scroll
	loadOnce sync.Once
	loaded   bool
}

func NewWikiTab() *container.TabItem {
	ui := &WikiUI{
		title: widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	}

	loadingLabel := widget.NewLabel("Loading wiki content...")
	icon := widget.NewIcon(theme.GridIcon())
	initialContent := container.NewCenter(container.NewHBox(loadingLabel, icon))

	ui.tab = container.NewTabItemWithIcon("Wiki", icons.ResourceWikiPng, initialContent)
	ui.setupLazyLoading()

	return ui.tab
}

func (ui *WikiUI) setupLazyLoading() {
	originalContent := ui.tab.Content
	ui.tab.Content = &lazyLoadContainer{
		content: originalContent,
		onShow:  ui.handleFirstShow,
	}
}

func (ui *WikiUI) handleFirstShow() {
	if !ui.loaded {
		go ui.loadPages()
	}
}

type lazyLoadContainer struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onShow  func()
	shown   bool
}

func (l *lazyLoadContainer) CreateRenderer() fyne.WidgetRenderer {
	if !l.shown {
		l.shown = true
		fyne.Do(l.onShow)
	}
	return widget.NewSimpleRenderer(l.content)
}

func (ui *WikiUI) loadPages() {
	ui.loadOnce.Do(func() {
		files, err := wikiFS.ReadDir("pages")
		if err != nil {
			fmt.Println("Error reading wiki dir:", err)
			return
		}

		var pages []Page
		for _, f := range files {
			title := strings.TrimSuffix(f.Name(), ".md")
			title = strings.ReplaceAll(title, "_", " ")
			page := Page{
				FileName: f.Name(),
				Title:    strings.Title(title),
			}
			if f.Name() == "home.md" {
				pages = append([]Page{page}, pages...)
			} else {
				pages = append(pages, page)
			}
		}
		ui.pages = pages

		if len(ui.pages) == 0 {
			fyne.Do(func() {
				ui.tab.Content = widget.NewLabel("No wiki content found")
			})
			return
		}

		ui.loaded = true
		ui.loadPage(0)
	})
}

func (ui *WikiUI) loadPage(idx int) {
	ui.showLoadingIndicator()
	go func() {
		page := &ui.pages[idx]
		if !ui.loadPageContent(page) {
			return
		}
		contentObjs := ui.parsePageContent(page.Content)
		fyne.Do(func() {
			ui.renderPage(idx, page, contentObjs)
		})
	}()
}

func (ui *WikiUI) showLoadingIndicator() {
	fyne.Do(func() {
		loadingLabel := widget.NewLabel("Loading page content...")
		icon := widget.NewIcon(theme.GridIcon())
		ui.tab.Content = container.NewCenter(container.NewHBox(loadingLabel, icon))
	})
}

func (ui *WikiUI) loadPageContent(page *Page) bool {
	if page.Content != "" {
		return true
	}
	data, err := wikiFS.ReadFile("pages/" + page.FileName)
	if err != nil {
		fmt.Println("Error reading file:", page.FileName, err)
		return false
	}
	page.Content = string(data)
	return true
}

func (ui *WikiUI) parsePageContent(content string) []fyne.CanvasObject {
	var contentObjs []fyne.CanvasObject
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if ui.processImageLine(line, &contentObjs) {
			continue
		}
		contentObjs = append(contentObjs, widget.NewRichTextFromMarkdown(line))
	}
	return contentObjs
}

func (ui *WikiUI) processImageLine(line string, contentObjs *[]fyne.CanvasObject) bool {
	if !strings.Contains(line, "{{image:") || !strings.Contains(line, "}}") {
		return false
	}

	start := strings.Index(line, "{{image:")
	end := strings.Index(line, "}}")
	if start < 0 || end <= start {
		return false
	}

	imgName := strings.TrimSpace(line[start+len("{{image:") : end])
	img := LoadImage(imgName)
	if img != nil {
		*contentObjs = append(*contentObjs, img)
	}

	ui.addTextAroundImage(line, start, end, contentObjs)
	return true
}

func (ui *WikiUI) addTextAroundImage(line string, start, end int, contentObjs *[]fyne.CanvasObject) {
	before := strings.TrimSpace(line[:start])
	after := strings.TrimSpace(line[end+2:])
	if before != "" {
		*contentObjs = append(*contentObjs, widget.NewRichTextFromMarkdown(before))
	}
	if after != "" {
		*contentObjs = append(*contentObjs, widget.NewRichTextFromMarkdown(after))
	}
}

func (ui *WikiUI) renderPage(idx int, page *Page, contentObjs []fyne.CanvasObject) {
	ui.index = idx
	ui.title.SetText(page.Title)

	nav := ui.createNavigation()
	scrollContent := ui.createScrollContent(page, contentObjs)

	ui.scroll = container.NewScroll(scrollContent)
	ui.tab.Content = container.NewBorder(ui.title, nav, nil, nil, ui.scroll)
}

func (ui *WikiUI) createNavigation() *fyne.Container {
	ui.btnPrev = widget.NewButton("Previous", func() {
		if ui.index > 0 {
			ui.loadPage(ui.index - 1)
		}
	})

	btnHome := widget.NewButtonWithIcon("", theme.HomeIcon(), func() {
		for i, p := range ui.pages {
			if p.FileName == "home.md" {
				ui.loadPage(i)
				break
			}
		}
	})

	ui.btnNext = widget.NewButton("Next", func() {
		if ui.index < len(ui.pages)-1 {
			ui.loadPage(ui.index + 1)
		}
	})

	pageCounter := widget.NewLabel(fmt.Sprintf("%d/%d", ui.index+1, len(ui.pages)))
	pageCounter.Alignment = fyne.TextAlignCenter

	return container.NewCenter(
		container.NewHBox(ui.btnPrev, btnHome, ui.btnNext, pageCounter),
	)
}

func (ui *WikiUI) createScrollContent(page *Page, contentObjs []fyne.CanvasObject) fyne.CanvasObject {
	if page.FileName == "home.md" {
		return ui.createHomePageContent(contentObjs)
	}
	return container.NewVBox(contentObjs...)
}

func (ui *WikiUI) createHomePageContent(contentObjs []fyne.CanvasObject) fyne.CanvasObject {
	links := []fyne.CanvasObject{}
	for i, p := range ui.pages {
		if p.FileName == "home.md" {
			continue
		}
		btn := widget.NewButton(p.Title, func(i int) func() {
			return func() { ui.loadPage(i) }
		}(i))
		links = append(links, btn)
	}
	return container.NewVBox(
		container.NewVBox(contentObjs...),
		container.NewCenter(container.NewVBox(links...)),
	)
}

func LoadImage(name string) *canvas.Image {
	if name == "" {
		return nil
	}
	data, err := imageFS.ReadFile("images/" + name)
	if err != nil {
		fmt.Println("Failed to load image:", name, "->", err)
		return nil
	}
	img := canvas.NewImageFromResource(fyne.NewStaticResource(name, data))
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(400, 300))
	return img
}
