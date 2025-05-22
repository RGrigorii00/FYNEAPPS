package ui

import (
	"FYNEAPPS/resources"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CustomDialog представляет кастомное диалоговое окно
type CustomDialog struct {
	parent    fyne.Window
	overlay   *widget.PopUp
	onConfirm func()
	onDismiss func()
}

// NewDialog создает новое диалоговое окно
func NewDialog(title, message string, parent fyne.Window) *CustomDialog {
	d := &CustomDialog{parent: parent}

	// Создаем контент диалога
	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	messageLabel := widget.NewLabel(message)
	messageLabel.Wrapping = fyne.TextWrapWord
	messageLabel.Alignment = fyne.TextAlignCenter

	okButton := widget.NewButton("OK", func() {
		d.Hide()
		if d.onConfirm != nil {
			d.onConfirm()
		}
	})

	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		layout.NewSpacer(),
		container.NewCenter(okButton),
	)

	// Создаем карточку диалога с рамкой
	dialogCard := container.NewStack(
		canvas.NewRectangle(theme.BackgroundColor()),
		container.NewPadded(content),
	)

	// Создаем PopUp без overlay
	d.overlay = widget.NewModalPopUp(
		container.NewCenter(dialogCard),
		parent.Canvas(),
	)

	// Устанавливаем размер диалога
	dialogSize := fyne.NewSize(300, 150)
	dialogCard.Resize(dialogSize)
	d.overlay.Resize(dialogSize)

	return d
}

// Show отображает диалоговое окно
func (d *CustomDialog) Show() {
	d.overlay.Show()
}

// Hide скрывает диалоговое окно
func (d *CustomDialog) Hide() {
	d.overlay.Hide()
	if d.onDismiss != nil {
		d.onDismiss()
	}
}

// SetOnConfirm устанавливает callback для кнопки OK
func (d *CustomDialog) SetOnConfirm(callback func()) {
	d.onConfirm = callback
}

// SetOnDismiss устанавливает callback при закрытии окна
func (d *CustomDialog) SetOnDismiss(callback func()) {
	d.onDismiss = callback
}

// ShowErrorDialog отображает диалог с ошибкой
func ShowErrorDialog(message string, parent fyne.Window) {
	d := NewDialog("Ошибка", message, parent)
	d.Show()
}

// ShowSuccessDialog отображает диалог с успешным выполнением
func ShowSuccessDialog(message string, parent fyne.Window) {
	d := NewDialog("Успешно", message, parent)
	d.Show()
}

func CreateLoginWindow(app fyne.App, onLogin func(string, string) bool) fyne.Window {
	loginWindow := app.NewWindow("ПГАТУ инфраструктура")
	loginWindow.SetFixedSize(true)
	loginWindow.Resize(fyne.NewSize(400, 400))
	loginWindow.CenterOnScreen()
	loginWindow.SetIcon(resources.ResourcePgatulogosmallPng)

	// Обработчик закрытия окна (крестик)
	loginWindow.SetCloseIntercept(func() {
		loginWindow.Hide() // Скрываем окно вместо закрытия
	})

	// Остальной код создания интерфейса остается без изменений
	title := widget.NewLabel("ПГАТУ Инфраструктура")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	logo := canvas.NewImageFromResource(resources.ResourcePgatulogosmallPng)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(150, 150))

	username := widget.NewEntry()
	username.SetPlaceHolder("Логин")
	username.SetText("user")
	username.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Введите логин")
		}
		return nil
	}

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Пароль")
	password.SetText("user")
	password.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Введите пароль")
		}
		return nil
	}

	loginButton := widget.NewButton("Войти", func() {
		if err := username.Validate(); err != nil {
			ShowErrorDialog(err.Error(), loginWindow)
			return
		}
		if err := password.Validate(); err != nil {
			ShowErrorDialog(err.Error(), loginWindow)
			return
		}
		if onLogin(username.Text, password.Text) {
			ShowSuccessDialog("Успешный вход в систему", loginWindow)
			loginWindow.Hide()
		} else {
			ShowErrorDialog("Неверный логин или пароль", loginWindow)
		}
	})

	// Кнопка Отмена с перезапуском окна
	cancelButton := widget.NewButton("Отмена", func() {
		loginWindow.Close()
		newLoginWindow := CreateLoginWindow(app, onLogin)
		newLoginWindow.Show()
	})

	form := container.NewVBox(
		container.NewPadded(username),
		container.NewPadded(password),
	)

	buttons := container.NewHBox(
		layout.NewSpacer(),
		cancelButton,
		loginButton,
		layout.NewSpacer(),
	)

	content := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		layout.NewSpacer(),
		form,
		layout.NewSpacer(),
		buttons,
		layout.NewSpacer(),
		container.NewCenter(widget.NewLabel("ПГАТУ Инфраструктура v.0.0.16")),
	)

	loginWindow.SetContent(container.NewPadded(content))
	return loginWindow
}
