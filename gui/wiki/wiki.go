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
	fyne.Do(func() {
		loadingLabel := widget.NewLabel("Loading page content...")
		icon := widget.NewIcon(theme.GridIcon())
		ui.tab.Content = container.NewCenter(container.NewHBox(loadingLabel, icon))
	})

	go func() {
		page := &ui.pages[idx]

		if page.Content == "" {
			data, err := wikiFS.ReadFile("pages/" + page.FileName)
			if err != nil {
				fmt.Println("Error reading file:", page.FileName, err)
				return
			}
			page.Content = string(data)
		}

		var contentObjs []fyne.CanvasObject
		lines := strings.Split(page.Content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "{{image:") && strings.Contains(line, "}}") {
				start := strings.Index(line, "{{image:")
				end := strings.Index(line, "}}")
				if start >= 0 && end > start {
					imgName := strings.TrimSpace(line[start+len("{{image:") : end])
					img := LoadImage(imgName)
					if img != nil {
						contentObjs = append(contentObjs, img)
					}

					before := strings.TrimSpace(line[:start])
					after := strings.TrimSpace(line[end+2:])
					if before != "" {
						contentObjs = append(contentObjs, widget.NewRichTextFromMarkdown(before))
					}
					if after != "" {
						contentObjs = append(contentObjs, widget.NewRichTextFromMarkdown(after))
					}
					continue
				}
			}
			contentObjs = append(contentObjs, widget.NewRichTextFromMarkdown(line))
		}

		fyne.Do(func() {
			ui.index = idx
			ui.title.SetText(page.Title)

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

			nav := container.NewCenter(
				container.NewHBox(ui.btnPrev, btnHome, ui.btnNext, pageCounter),
			)

			var scrollContent fyne.CanvasObject
			if page.FileName == "home.md" {
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
				scrollContent = container.NewVBox(
					container.NewVBox(contentObjs...),
					container.NewCenter(container.NewVBox(links...)),
				)
			} else {
				scrollContent = container.NewVBox(contentObjs...)
			}

			ui.scroll = container.NewScroll(scrollContent)
			ui.tab.Content = container.NewBorder(ui.title, nav, nil, nil, ui.scroll)
		})
	}()
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
