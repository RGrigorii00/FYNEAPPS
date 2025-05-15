package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CustomDialog представляет кастомное диалоговое окно
type CustomDialog struct {
	window    fyne.Window
	onDismiss func()
}

// NewErrorDialog создает диалог для отображения ошибок
func NewErrorDialog(title, message string, app fyne.App) *CustomDialog {
	d := &CustomDialog{}

	d.window = app.NewWindow(title)
	d.window.SetFixedSize(true)
	d.window.Resize(fyne.NewSize(300, 150))
	d.window.CenterOnScreen()

	content := container.NewVBox(
		container.NewHBox(
			widget.NewIcon(theme.ErrorIcon()),
			widget.NewLabel(message),
		),
		layout.NewSpacer(),
		container.NewCenter(
			widget.NewButton("OK", func() {
				d.window.Hide()
				if d.onDismiss != nil {
					d.onDismiss()
				}
			}),
		),
	)

	d.window.SetContent(content)
	return d
}

// CreateLoginWindow создает красивое окно входа в систему
func CreateLoginWindow(a fyne.App, onLogin func(string, string) bool) fyne.Window {
	loginWindow := a.NewWindow("Вход в систему")
	loginWindow.SetFixedSize(true)
	loginWindow.Resize(fyne.NewSize(400, 300))
	loginWindow.CenterOnScreen()

	// Основные элементы интерфейса
	title := widget.NewLabel("ПГАТУ Инфраструктура")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	logo := widget.NewIcon(theme.ComputerIcon())
	logo.Resize(fyne.NewSize(64, 64))

	username := widget.NewEntry()
	username.SetPlaceHolder("Логин")
	username.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Введите логин")
		}
		return nil
	}

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Пароль")
	password.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Введите пароль")
		}
		return nil
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "", Widget: username},
			{Text: "", Widget: password},
		},
		SubmitText: "Войти",
		CancelText: "Отмена",
		OnSubmit: func() {
			if err := username.Validate(); err != nil {
				// NewErrorDialog("Ошибка", err.Error(), a).Show()
				return
			}

			if err := password.Validate(); err != nil {
				// NewErrorDialog("Ошибка", err.Error(), a).Show()
				return
			}

			if onLogin(username.Text, password.Text) {
				loginWindow.Hide()
			} else {
				// NewErrorDialog(
				// 	"Ошибка входа",
				// 	"Неверный логин или пароль",
				// 	a,
				// ).Show()
			}
		},
		OnCancel: func() {
			// a.Quit()
		},
	}

	// Компоновка интерфейса
	content := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		layout.NewSpacer(),
		form,
		layout.NewSpacer(),
		widget.NewLabel("© ПГАТУ"),
	)

	loginWindow.SetContent(content)
	return loginWindow
}
