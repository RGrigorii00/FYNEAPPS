package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func CreateLoginWindow(a fyne.App, onLogin func(string, string) bool) fyne.Window {
	loginWindow := a.NewWindow("Вход в систему")
	loginWindow.SetFixedSize(true)
	loginWindow.Resize(fyne.NewSize(300, 200))

	username := widget.NewEntry()
	username.SetPlaceHolder("Логин")

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Пароль")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Логин", Widget: username},
			{Text: "Пароль", Widget: password},
		},
		OnSubmit: func() {
			if onLogin(username.Text, password.Text) {
				loginWindow.Hide()
			} else {
				dialog.ShowError(fmt.Errorf("Неверные учетные данные"), loginWindow)
			}
		},
	}

	loginWindow.SetContent(container.NewVBox(
		widget.NewLabel("ПГАТУ Инфраструктура"),
		form,
	))

	return loginWindow
}
