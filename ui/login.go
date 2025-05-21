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

// CreateLoginWindow создает окно входа в систему
func CreateLoginWindow(app fyne.App, onLogin func(string, string) bool) fyne.Window {
	loginWindow := app.NewWindow("ПГАТУ инфраструктура")
	loginWindow.SetFixedSize(true)
	loginWindow.Resize(fyne.NewSize(400, 300))
	loginWindow.CenterOnScreen()
	loginWindow.SetIcon(resources.ResourcePgatulogosmallPng)

	// Основные элементы интерфейса
	title := widget.NewLabel("ПГАТУ Инфраструктура")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	logo := widget.NewIcon(resources.ResourcePgatulogosmallPng)
	logo.Resize(fyne.NewSize(512, 512))

	username := widget.NewEntry()
	username.SetPlaceHolder("Логин")
	username.SetText("user") // Предзаполняем логин
	username.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Введите логин")
		}
		return nil
	}

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Пароль")
	password.SetText("user") // Предзаполняем пароль
	password.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Введите пароль")
		}
		return nil
	}

	loginButton := widget.NewButton("Войти", func() {
		// Валидация логина
		if err := username.Validate(); err != nil {
			ShowErrorDialog(err.Error(), loginWindow)
			return
		}

		// Валидация пароля
		if err := password.Validate(); err != nil {
			ShowErrorDialog(err.Error(), loginWindow)
			return
		}

		// Проверка учетных данных
		if onLogin(username.Text, password.Text) {
			ShowSuccessDialog("Успешный вход в систему", loginWindow)
			loginWindow.Hide()
		} else {
			ShowErrorDialog("Неверный логин или пароль", loginWindow)
		}
	})

	cancelButton := widget.NewButton("Отмена", func() {
		loginWindow.Close()
	})

	form := container.NewVBox(
		container.NewPadded(username),
		container.NewPadded(password),
		container.NewHBox(
			layout.NewSpacer(),
			container.NewPadded(cancelButton),
			container.NewPadded(loginButton),
			layout.NewSpacer(),
		),
	)

	// Компоновка интерфейса
	content := container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		layout.NewSpacer(),
		form,
		layout.NewSpacer(),
		container.NewCenter(widget.NewLabel("© ПГАТУ")),
	)

	loginWindow.SetContent(content)
	return loginWindow
}
