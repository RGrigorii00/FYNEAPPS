package tabs

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AppDefinition struct {
	ID          string
	Name        string
	Description string
	IconPath    string
	Version     string
	Category    string
}

func CreateAppsLibraryTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Библиотека приложений ПГАТУ", theme.Color(theme.ColorNameForeground))
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	appDefinitions := []AppDefinition{
		{
			ID:          "app1",
			Name:        "Text Editor",
			Description: "Редактор текста с подсветкой синтаксиса",
			IconPath:    "images/main_screen/pgatu_logo_small.png",
			Version:     "1.2.4",
			Category:    "Development",
		},
		{
			ID:          "app2",
			Name:        "Image Viewer",
			Description: "Просмотр и базовое редактирование изображений",
			IconPath:    "images/main_screen/pgatu_logo_small.png",
			Version:     "3.0.1",
			Category:    "Graphics",
		},
		{
			ID:          "app3",
			Name:        "System Monitor",
			Description: "Мониторинг системных ресурсов",
			IconPath:    "images/main_screen/pgatu_logo_small.png",
			Version:     "2.5.0",
			Category:    "Utilities",
		},
		{
			ID:          "app4",
			Name:        "Media Player",
			Description: "Воспроизведение аудио и видео",
			IconPath:    "images/main_screen/pgatu_logo_small.png",
			Version:     "4.1.2",
			Category:    "Multimedia",
		},
	}

	scrollContainer := container.NewVScroll(nil)
	scrollContainer.SetMinSize(fyne.NewSize(800, 600))

	updateContent := func() {
		grid := container.NewGridWithColumns(3)

		for i := 0; i < len(appDefinitions); i++ {
			card, err := createAppCard(appDefinitions[i])
			if err != nil {
				log.Printf("Error creating card: %v", err)
				continue
			}
			grid.Add(container.NewPadded(card))
		}

		for i := len(appDefinitions); i%3 != 0; i++ {
			grid.Add(container.NewPadded(widget.NewLabel("")))
		}

		scrollContainer.Content = container.NewVBox(
			container.NewPadded(title),
			widget.NewSeparator(),
			grid,
		)
	}

	updateContent()

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), updateContent)
	refreshBtn.Importance = widget.MediumImportance

	return container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), refreshBtn, layout.NewSpacer()),
		nil,
		nil,
		scrollContainer,
	)
}

func createAppCard(appDef AppDefinition) (fyne.CanvasObject, error) {
	icon, err := loadAppIcon(appDef.IconPath)
	if err != nil {
		return nil, fmt.Errorf("could not load icon: %v", err)
	}
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(128, 128))

	nameLabel := widget.NewLabel(appDef.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel.Alignment = fyne.TextAlignCenter

	descLabel := widget.NewLabel(appDef.Description)
	descLabel.Wrapping = fyne.TextWrapWord
	descLabel.Alignment = fyne.TextAlignCenter

	versionLabel := widget.NewLabel(fmt.Sprintf("Версия: %s", appDef.Version))
	versionLabel.Alignment = fyne.TextAlignCenter

	categoryLabel := widget.NewLabel(fmt.Sprintf("Категория: %s", appDef.Category))
	categoryLabel.Alignment = fyne.TextAlignCenter

	launchBtn := widget.NewButtonWithIcon("Запуск", theme.MediaPlayIcon(), nil)
	downloadBtn := widget.NewButtonWithIcon("Скачать", theme.DownloadIcon(), nil)
	updateBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)

	buttons := container.NewHBox(
		layout.NewSpacer(),
		downloadBtn,
		updateBtn,
		launchBtn,
		layout.NewSpacer(),
	)

	cardContent := container.NewVBox(
		container.NewCenter(icon),
		container.NewCenter(nameLabel),
		container.NewPadded(descLabel),
		container.NewCenter(versionLabel),
		container.NewCenter(categoryLabel),
		layout.NewSpacer(),
		buttons,
	)

	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	cardBackground.CornerRadius = 12
	cardBackground.StrokeColor = theme.Color(theme.ColorNameSeparator)
	cardBackground.StrokeWidth = 1

	shadow := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 30})
	shadow.CornerRadius = 12

	card := container.NewStack(
		container.NewPadded(shadow),
		container.NewStack(
			cardBackground,
			container.NewPadded(cardContent),
		),
	)

	return container.NewPadded(card), nil
}

func loadAppIcon(path string) (*canvas.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, _, err = image.Decode(file)
	if err != nil {
		return nil, err
	}

	return canvas.NewImageFromFile(path), nil
}
