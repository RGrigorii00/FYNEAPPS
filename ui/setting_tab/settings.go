package settings

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// AppSettings содержит настройки приложения
type AppSettings struct {
	Theme      string `json:"theme"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Fullscreen bool   `json:"fullscreen"`
	Language   string `json:"language"` // "ru" или "en"
}

// Локализованные строки
var locales = map[string]map[string]string{
	"ru": {
		"MyComputer":   "Мой компьютер",
		"AppsLibrary":  "Библиотека приложений",
		"Processes":    "Процессы компьютера",
		"ServerStatus": "Статус серверов ПГАТУ",
		"Portal":       "Портал ПГАТУ",
		"Site":         "Сайт ПГАТУ",
		"Update":       "Обновить приложение",
		"Repository":   "Последние обновления",
		"Settings":     "Настройки приложения",
		"title":        "Настройки приложения",
		"theme":        "Тема:",
		"light":        "Светлая",
		"dark":         "Тёмная",
		"system":       "Системная",
		"resolution":   "Разрешение:",
		"fullscreen":   "Полноэкранный режим",
		"apply":        "Применить",
		"success":      "Успешно",
		"saved":        "Настройки сохранены",
		"language":     "Язык:",
		"language_ru":  "Русский",
		"language_en":  "English",
	},
	"en": {
		"MyComputer":   "My computer",
		"AppsLibrary":  "Apps Library",
		"Processes":    "Computer Processes",
		"ServerStatus": "PGATU Server Status",
		"Portal":       "PGATU Portal",
		"Site":         "PGATU Site",
		"Update":       "Update Application",
		"Repository":   "Latest Updates",
		"Settings":     "Application Settings",
		"title":        "Application Settings",
		"theme":        "Theme:",
		"light":        "Light",
		"dark":         "Dark",
		"system":       "System",
		"resolution":   "Resolution:",
		"fullscreen":   "Fullscreen mode",
		"apply":        "Apply",
		"success":      "Success",
		"saved":        "Settings saved",
		"language":     "Language:",
		"language_ru":  "Russian",
		"language_en":  "English",
	},
}

// Добавляем глобальную переменную для хранения callback'ов обновления
var LanguageChangeCallbacks []func()

// Добавляем функцию для подписки на изменения языка
func OnLanguageChange(callback func()) {
	LanguageChangeCallbacks = append(LanguageChangeCallbacks, callback)
}

// currentLanguage хранит текущий язык
var currentLanguage = "ru"

// GetLocalizedString возвращает локализованную строку
func GetLocalizedString(key string) string {
	return locales[currentLanguage][key]
}

func CreateSettingsTab(window fyne.Window, myApp fyne.App) fyne.CanvasObject {
	// Загрузка текущих настроек
	currentSettings := LoadSettings(myApp, window)
	currentLanguage = currentSettings.Language // Устанавливаем текущий язык

	// Элементы UI
	themeSelect := widget.NewSelect(
		[]string{GetLocalizedString("light"), GetLocalizedString("dark"), GetLocalizedString("system")},
		func(s string) { currentSettings.Theme = s },
	)
	themeSelect.SetSelected(currentSettings.Theme)

	resolutions := []string{"800x600", "1024x768", "1280x720", "1920x1080"}
	resolutionSelect := widget.NewSelect(resolutions, func(s string) {
		parts := strings.Split(s, "x")
		if len(parts) == 2 {
			currentSettings.Width, _ = strconv.Atoi(parts[0])
			currentSettings.Height, _ = strconv.Atoi(parts[1])
		}
	})
	resolutionSelect.SetSelected(strconv.Itoa(currentSettings.Width) + "x" + strconv.Itoa(currentSettings.Height))

	fullscreenCheck := widget.NewCheck(GetLocalizedString("fullscreen"), func(b bool) {
		currentSettings.Fullscreen = b
	})
	fullscreenCheck.SetChecked(currentSettings.Fullscreen)

	// В функции выбора языка добавляем вызов callback'ов
	languageSelect := widget.NewSelect(
		[]string{GetLocalizedString("language_ru"), GetLocalizedString("language_en")},
		func(s string) {
			if s == GetLocalizedString("language_ru") {
				currentSettings.Language = "ru"
			} else {
				currentSettings.Language = "en"
			}
			currentLanguage = currentSettings.Language
			updateUIElements(themeSelect, fullscreenCheck, resolutionSelect)

			// Вызываем все зарегистрированные callback'и
			for _, callback := range LanguageChangeCallbacks {
				callback()
			}
		},
	)

	if currentSettings.Language == "ru" {
		languageSelect.SetSelected(GetLocalizedString("language_ru"))
	} else {
		languageSelect.SetSelected(GetLocalizedString("language_en"))
	}

	applyBtn := widget.NewButton(GetLocalizedString("apply"), func() {
		applySettings(currentSettings, myApp, window)
		if err := saveSettings(currentSettings); err != nil {
			dialog.ShowError(err, window)
		} else {
			dialog.ShowInformation(GetLocalizedString("success"), GetLocalizedString("saved"), window)
		}
	})

	// Компоновка интерфейса
	return container.NewVBox(
		widget.NewLabelWithStyle(GetLocalizedString("title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem(GetLocalizedString("language"), languageSelect),
			widget.NewFormItem(GetLocalizedString("theme"), themeSelect),
			widget.NewFormItem(GetLocalizedString("resolution"), resolutionSelect),
			widget.NewFormItem("", fullscreenCheck),
		),
		layout.NewSpacer(),
		container.NewCenter(applyBtn),
	)
}

// updateUIElements обновляет тексты элементов при изменении языка
func updateUIElements(themeSelect *widget.Select, fullscreenCheck *widget.Check, resolutionSelect *widget.Select) {
	// Обновляем варианты тем
	themeSelect.Options = []string{GetLocalizedString("light"), GetLocalizedString("dark"), GetLocalizedString("system")}
	themeSelect.Refresh()

	// Обновляем текст чекбокса
	fullscreenCheck.Text = GetLocalizedString("fullscreen")
	fullscreenCheck.Refresh()

	// Здесь можно добавить обновление других элементов по необходимости
}

func LoadSettings(myApp fyne.App, window fyne.Window) AppSettings {
	defaultSettings := AppSettings{
		Theme:      GetLocalizedString("light"),
		Width:      800,
		Height:     600,
		Fullscreen: false,
		Language:   "ru", // Язык по умолчанию
	}

	data, err := os.ReadFile("settings.json")
	if err != nil {
		return defaultSettings
	}

	var settings AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings
	}

	return settings
}

func saveSettings(settings AppSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("settings.json", data, 0644)
}

func applySettings(settings AppSettings, app fyne.App, window fyne.Window) {
	// Применение темы
	switch settings.Theme {
	case GetLocalizedString("dark"):
		app.Settings().SetTheme(theme.DarkTheme())
	case GetLocalizedString("system"):
		app.Settings().SetTheme(theme.DefaultTheme())
	default:
		app.Settings().SetTheme(theme.LightTheme())
	}

	// Размер окна
	window.Resize(fyne.NewSize(float32(settings.Width), float32(settings.Height)))

	// Полноэкранный режим
	window.SetFullScreen(settings.Fullscreen)

	// Обновляем текущий язык
	currentLanguage = settings.Language
}
