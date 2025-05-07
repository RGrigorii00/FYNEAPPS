package main

import (
	tabs "FYNEAPPS/ui"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

func main() {
	// Создаём новое приложение
	myApp := app.New()

	// Загружаем иконку из файла
	icon, err := fyne.LoadResourceFromPath("images/icons/pgatu_logo_small.png")
	if err != nil {
		// Если не удалось загрузить иконку, используем стандартную
		fyne.LogError("Не удалось загрузить иконку", err)
	} else {
		myApp.SetIcon(icon) // Устанавливаем иконку для всего приложения
	}

	// Создаём главное окно
	window := myApp.NewWindow("ПГАТУ Инфраструктура")
	window.Resize(fyne.NewSize(800, 600))

	// Инициализация менеджера настроек
	// settingsManager := settings.NewSettingsManager(myApp, window)

	// Проверяем, поддерживается ли системный трей
	if desk, ok := myApp.(desktop.App); ok {
		// Создаем меню для трея

		if runtime.GOOS == "windows" {
			m := fyne.NewMenu("ПГАТУ Инфраструктура",
				fyne.NewMenuItem("Развернуть", func() {
					window.Show()
				}),
				fyne.NewMenuItem("123", func() {
					window.Show()
				}),
				fyne.NewMenuItem("Выход", func() {
					myApp.Quit()
				}),
			)

			desk.SetSystemTrayMenu(m)

			// Устанавливаем иконку для трея
			if icon != nil {
				desk.SetSystemTrayIcon(icon)
			}
		}
	}

	// Обработка сворачивания в трей
	window.SetCloseIntercept(func() {
		window.Hide() // Скрываем окно вместо закрытия
	})

	// Если иконка загружена, устанавливаем её для окна
	if icon != nil {
		window.SetIcon(icon)
	}

	// Создаем все вкладки
	tabs := tabs.CreateAppTabs(myApp, window)

	// Устанавливаем вкладки как содержимое окна
	window.SetContent(tabs)

	// Запускаем приложение
	window.ShowAndRun()
}
