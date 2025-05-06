package tabs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func CreateMemoryTab(window fyne.Window) fyne.CanvasObject {
	title := widget.NewLabel("Memory Information")
	title.TextStyle = fyne.TextStyle{Bold: true}

	content := container.NewVBox(
		title,
		widget.NewLabel("Memory information will be displayed here"),
	)

	return content
}
