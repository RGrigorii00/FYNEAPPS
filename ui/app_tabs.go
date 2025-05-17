package ui

import (
	"FYNEAPPS/database"
	"FYNEAPPS/resources"
	settings "FYNEAPPS/ui/setting_tab"
	"FYNEAPPS/ui/tabs"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// GitHubRelease представляет структуру ответа GitHub API о релизах
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Создаем переменные для хранения элементов, которые нужно обновлять
var (
	cpuBtn, appslibraryBtn, processBtn     *widget.Button
	serverstatusBtn, portalBtn, siteBtn    *widget.Button
	updateBtn, repositoriiBtn, settingsBtn *widget.Button
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
	LoggedIn  bool      `json:"logged_in"`
}

// Добавьте глобальные переменные для управления состоянием вкладки
var (
	ticketsTab     fyne.CanvasObject
	ticketsCleanup func()
)

func CreateAppTabs(myApp fyne.App, window fyne.Window) fyne.CanvasObject {
	// Создаем кнопки с иконками для вертикального меню

	cpuBtn := widget.NewButtonWithIcon(settings.GetLocalizedString("MyComputer"), theme.ComputerIcon(), nil)
	appslibraryBtn := widget.NewButtonWithIcon("Библиотека приложений", theme.SearchIcon(), nil)
	processBtn := widget.NewButtonWithIcon("Процессы компьютера", theme.ListIcon(), nil)
	serverstatusBtn := widget.NewButtonWithIcon("Статус серверов ПГАТУ", theme.StorageIcon(), nil)
	compterprogramsBtn := widget.NewButtonWithIcon("Программы на компьютере", theme.StorageIcon(), nil)
	ticketBtn := widget.NewButtonWithIcon("Тикеты", theme.StorageIcon(), nil)

	// Создаем кастомную кнопку
	portalBtn := widget.NewButton("", nil)
	portalBtn.SetText("Портал ПГАТУ")
	portalBtn.OnTapped = func() {
		openBrowser("https://portal.pgatu.ru/")
	}
	siteBtn := widget.NewButtonWithIcon("Сайт ПГАТУ", theme.SettingsIcon(), func() {
		openBrowser("https://pgatu.ru/today/")
	})
	repositoriiBtn := widget.NewButtonWithIcon("Последние обновления", theme.SettingsIcon(), func() {
		openBrowser("https://github.com/RGrigorii00/FYNEAPPS/releases")
	})

	// Добавляем новую кнопку обновления
	updateBtn := widget.NewButtonWithIcon("Обновить приложение", theme.ViewRefreshIcon(), func() {
		updateApp(window)
	})

	// Добавляем новую кнопку обновления
	settingsBtn := widget.NewButtonWithIcon("Настройки приложения", theme.SettingsIcon(), func() {
		updateApp(window)
	})

	// Настраиваем стиль кнопок
	buttons := []*widget.Button{cpuBtn, appslibraryBtn, processBtn, serverstatusBtn, compterprogramsBtn, ticketBtn, portalBtn, siteBtn, updateBtn, repositoriiBtn, settingsBtn}
	for _, btn := range buttons {
		btn.Alignment = widget.ButtonAlignLeading
		btn.Importance = widget.MediumImportance
	}

	// Функция для обновления текстов
	updateUI := func() {
		cpuBtn.SetText(settings.GetLocalizedString("MyComputer"))
		appslibraryBtn.SetText(settings.GetLocalizedString("AppsLibrary"))
		processBtn.SetText(settings.GetLocalizedString("Processes"))
		serverstatusBtn.SetText(settings.GetLocalizedString("ServerStatus"))
		portalBtn.SetText(settings.GetLocalizedString("Portal"))
		siteBtn.SetText(settings.GetLocalizedString("Site"))
		updateBtn.SetText(settings.GetLocalizedString("Update"))
		repositoriiBtn.SetText(settings.GetLocalizedString("Repository"))
		settingsBtn.SetText(settings.GetLocalizedString("Settings"))
	}

	// Подписываемся на изменения языка
	settings.OnLanguageChange(updateUI)

	// Загрузка изображения
	img := resources.ResourcePgatulogosmallPng // fyne.LoadResourceFromPath("images/main_screen/pgatu_logo_small.png")

	// Создаем изображение с возможностью управления размером
	image := canvas.NewImageFromResource(img)
	image.FillMode = canvas.ImageFillContain // Сохраняет пропорции
	image.SetMinSize(fyne.NewSize(150, 100)) // Устанавливаем минимальный размер (ширина, высота)

	systemGroup := container.NewVBox(
		widget.NewLabel("Системные инструменты:"),
		cpuBtn,
		appslibraryBtn,
		processBtn,
		compterprogramsBtn,
		ticketBtn,
	)

	webGroup := container.NewVBox(
		widget.NewLabel("Веб-ресурсы:"),
		serverstatusBtn,
		portalBtn,
		siteBtn,
	)

	appGroup := container.NewVBox(
		widget.NewLabel("Управление приложением:"),
		updateBtn,
		repositoriiBtn,
		settingsBtn,
	)

	menu := container.NewVBox(
		container.NewCenter(
			image,
		),
		container.NewCenter(
			widget.NewLabel("ПГАТУ Инфраструктура"),
		),
		widget.NewSeparator(),

		systemGroup,
		layout.NewSpacer(),
		webGroup,
		layout.NewSpacer(),
		appGroup,

		widget.NewSeparator(),
		container.NewCenter(
			widget.NewLabel("v0.0.12 alpha"),
		),
	)

	// Убираем стандартные отступы у VBox
	menu.Layout = layout.NewVBoxLayout()

	// Создаем контейнер для контента
	content := container.NewStack()
	currentTab := tabs.CreateHardwareTab(window) // Начальная вкладка
	content.Objects = []fyne.CanvasObject{currentTab}

	// Обновленная функция setActiveButton
	setActiveButton := func(activeBtn *widget.Button) {
		for _, btn := range buttons {
			if btn == activeBtn {
				btn.Importance = widget.HighImportance
				btn.Refresh() // Обновляем отображение кнопки
			} else {
				btn.Importance = widget.MediumImportance
				btn.Refresh() // Обновляем отображение кнопки
			}
		}
	}

	// setTab := func(tab fyne.CanvasObject, btn *widget.Button) {
	// 	setActiveButton(btn)
	// 	content.Objects = []fyne.CanvasObject{tab}
	// 	content.Refresh()
	// }

	// Создаем подключение к базе данных
	db := database.New()
	opts := database.ConnectionOptions{
		Host:     "83.166.245.249",
		Port:     "5432",
		User:     "user",
		Password: "user",
		DBName:   "grafana_db",
		SSLMode:  "default",
	}

	// Устанавливаем соединение
	db.Connect(opts)
	defer db.Disconnect() // Закрываем соединение при выходе

	// Модифицированные обработчики для кнопок
	cpuBtn.OnTapped = func() {
		setActiveButton(cpuBtn)
		content.Objects = []fyne.CanvasObject{tabs.CreateHardwareTab(window)}
		content.Refresh()
	}

	appslibraryBtn.OnTapped = func() {
		setActiveButton(appslibraryBtn)
		content.Objects = []fyne.CanvasObject{tabs.CreateAppsLibraryTab(window, db)}
		content.Refresh()
	}

	processBtn.OnTapped = func() {
		setActiveButton(processBtn)
		content.Objects = []fyne.CanvasObject{tabs.CreateProcessesTab(window)}
		content.Refresh()
	}

	serverstatusBtn.OnTapped = func() {
		setActiveButton(serverstatusBtn)
		content.Objects = []fyne.CanvasObject{tabs.CreateServerStatusTab(window)}
		content.Refresh()
	}

	compterprogramsBtn.OnTapped = func() {
		setActiveButton(compterprogramsBtn)
		content.Objects = []fyne.CanvasObject{tabs.CreateSoftwareTab(window)}
		content.Refresh()
	}

	ticketBtn.OnTapped = func() {
		setActiveButton(ticketBtn)

		// Очищаем предыдущую вкладку, если она существует
		if ticketsCleanup != nil {
			ticketsCleanup()
		}

		// Создаем новую вкладку
		tab, cleanup := tabs.CreateTicketsTab(window)
		ticketsCleanup = cleanup

		content.Objects = []fyne.CanvasObject{tab}
		content.Refresh()
	}

	settingsBtn.OnTapped = func() {
		setActiveButton(settingsBtn)
		content.Objects = []fyne.CanvasObject{settings.CreateSettingsTab(window, myApp)}
		content.Refresh()
	}

	// Устанавливаем активную кнопку по умолчанию
	setActiveButton(cpuBtn)

	// Основной контейнер с вертикальным меню слева
	return container.NewBorder(
		nil, nil,
		container.NewBorder(
			nil,
			nil,
			container.NewVBox(
				widget.NewLabel(""),
				menu,
				layout.NewSpacer(),
			),
			widget.NewSeparator(),
		),
		nil,
		content,
	)
}

// Добавьте функцию для очистки при закрытии приложения
func cleanupApp() {
	if ticketsCleanup != nil {
		ticketsCleanup()
	}
	// ... другие cleanup-функции ...
}

// updateApp проверяет и устанавливает обновления
func updateApp(window fyne.Window) {
	progress := widget.NewProgressBarInfinite()
	statusLabel := widget.NewLabel("Проверка обновлений...")

	dialog := widget.NewModalPopUp(
		container.NewVBox(
			statusLabel,
			progress,
		),
		window.Canvas(),
	)
	dialog.Show()

	go func() {
		defer dialog.Hide()

		// 1. Проверка текущей версии (замените на свою логику)
		currentVersion := "0.0.12" // Это должно быть из вашего приложения

		// 2. Получение информации о последнем релизе
		statusLabel.SetText("Получение информации о релизе...")
		release, err := getLatestRelease("yourusername", "yourrepository")
		if err != nil {
			fyne.LogError("Ошибка получения релиза", err)
			// widget.ShowError(fmt.Errorf("не удалось проверить обновления: %v", err), window)
			return
		}

		// 3. Проверка необходимости обновления
		if release.TagName == currentVersion {
			// widget.ShowInformation("Обновление", "У вас уже установлена последняя версия", window)
			return
		}

		// 4. Поиск подходящего билда для текущей ОС
		statusLabel.SetText("Поиск подходящего билда...")
		assetURL := ""
		osPattern := ""

		switch runtime.GOOS {
		case "windows":
			osPattern = "windows"
		case "darwin":
			osPattern = "darwin"
		case "linux":
			osPattern = "linux"
		}

		for _, asset := range release.Assets {
			if contains(asset.Name, osPattern) {
				assetURL = asset.URL
				break
			}
		}

		if assetURL == "" {
			// widget.ShowError(fmt.Errorf("не найден подходящий билд для вашей ОС"), window)
			return
		}

		// 5. Скачивание нового билда
		statusLabel.SetText("Скачивание обновления...")
		exePath, err := os.Executable()
		if err != nil {
			fyne.LogError("Ошибка получения пути", err)
			// widget.ShowError(fmt.Errorf("ошибка получения пути: %v", err), window)
			return
		}

		downloadPath := filepath.Join(filepath.Dir(exePath), "update_temp")
		err = downloadFile(downloadPath, assetURL)
		if err != nil {
			fyne.LogError("Ошибка загрузки", err)
			// widget.ShowError(fmt.Errorf("ошибка загрузки: %v", err), window)
			return
		}

		// 6. Установка обновления (это сложная часть)
		statusLabel.SetText("Установка обновления...")
		// Здесь должна быть логика замены текущего исполняемого файла
		// Это зависит от ОС и может потребовать дополнительных скриптов

		// Временное сообщение об успехе
		// widget.ShowInformation("Обновление",
		// 	fmt.Sprintf("Обновление до версии %s успешно загружено. Перезапустите приложение.", release.TagName),
		// 	window)
	}()
}

// getLatestRelease получает информацию о последнем релизе
func getLatestRelease(owner, repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка HTTP: %s", resp.Status)
	}

	var release GitHubRelease
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return nil, err
	}

	return &release, nil
}

// downloadFile скачивает файл по URL
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// contains проверяет наличие подстроки
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

// openBrowser открывает URL в браузере по умолчанию
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}

	err := exec.Command(cmd, args...).Start()
	if err != nil {
		fyne.LogError("Ошибка при открытии браузера", err)
	}
}
