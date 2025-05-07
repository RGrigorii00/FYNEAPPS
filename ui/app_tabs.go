package ui

import (
	"FYNEAPPS/ui/tabs"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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

func CreateAppTabs(myApp fyne.App, window fyne.Window) fyne.CanvasObject {
	// Создаем кнопки с иконками для вертикального меню

	cpuBtn := widget.NewButtonWithIcon(tabs.GetLocalizedString("MyComputer"), theme.ComputerIcon(), nil)
	appslibraryBtn := widget.NewButtonWithIcon("Библиотека приложений", theme.SearchIcon(), nil)
	processBtn := widget.NewButtonWithIcon("Процессы компьютера", theme.ListIcon(), nil)
	serverstatusBtn := widget.NewButtonWithIcon("Статус серверов ПГАТУ", theme.StorageIcon(), nil)
	// // Создаем изображение из ресурса
	// img := canvas.NewImageFromResource(resources.PortalIcon)
	// img.FillMode = canvas.ImageFillContain
	// img.SetMinSize(fyne.NewSize(32, 32)) // Настраиваем размер

	// Создаем кастомную кнопку
	portalBtn := widget.NewButton("", nil)
	portalBtn.SetText("Портал ПГАТУ")
	// portalBtn.SetIcon(resources.PortalIcon)
	portalBtn.OnTapped = func() {
		openBrowser("https://portal.pgatu.ru/")
	}
	siteBtn := widget.NewButtonWithIcon("Сайт ПГАТУ", theme.SettingsIcon(), func() {
		openBrowser("https://pgsha.ru/today/")
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
	buttons := []*widget.Button{cpuBtn, appslibraryBtn, processBtn, serverstatusBtn, portalBtn, siteBtn, updateBtn, repositoriiBtn, settingsBtn}
	for _, btn := range buttons {
		btn.Alignment = widget.ButtonAlignLeading
		btn.Importance = widget.MediumImportance
	}

	// Функция для обновления текстов
	updateUI := func() {
		cpuBtn.SetText(tabs.GetLocalizedString("MyComputer"))
		appslibraryBtn.SetText(tabs.GetLocalizedString("AppsLibrary"))
		processBtn.SetText(tabs.GetLocalizedString("Processes"))
		serverstatusBtn.SetText(tabs.GetLocalizedString("ServerStatus"))
		portalBtn.SetText(tabs.GetLocalizedString("Portal"))
		siteBtn.SetText(tabs.GetLocalizedString("Site"))
		updateBtn.SetText(tabs.GetLocalizedString("Update"))
		repositoriiBtn.SetText(tabs.GetLocalizedString("Repository"))
		settingsBtn.SetText(tabs.GetLocalizedString("Settings"))
	}

	// Подписываемся на изменения языка
	tabs.OnLanguageChange(updateUI)

	// Загрузка изображения
	img, err := fyne.LoadResourceFromPath("images/main_screen/pgatu_logo_small.png")
	if err != nil {
		log.Println("Error loading image:", err)
		img = nil
	}

	// Создаем изображение с возможностью управления размером
	image := canvas.NewImageFromResource(img)
	image.FillMode = canvas.ImageFillContain // Сохраняет пропорции
	image.SetMinSize(fyne.NewSize(150, 100)) // Устанавливаем минимальный размер (ширина, высота)

	menu := container.NewBorder(
		nil, // Верхний элемент (nil - нет элемента)
		nil, // Нижний элемент
		nil, // Левый элемент
		nil, // Правый элемент
		container.NewVBox(
			container.NewCenter(
				image,
			),
			container.NewCenter( // Обертка для центрирования текста
				widget.NewLabel("ПГАТУ Инфраструктура"),
			),
			widget.NewSeparator(),
			cpuBtn,
			appslibraryBtn,
			processBtn,
			serverstatusBtn,
			portalBtn,
			siteBtn,
			updateBtn,
			repositoriiBtn,
			settingsBtn,
			widget.NewSeparator(),
			container.NewCenter( // Обертка для центрирования текста
				widget.NewLabel("v0.0.1 alpha"),
			),
		),
	)

	// Убираем стандартные отступы у VBox
	menu.Layout = layout.NewVBoxLayout()

	// Создаем контейнер для контента
	content := container.NewStack()
	currentTab := tabs.CreateHardwareTab(window) // Начальная вкладка
	content.Objects = []fyne.CanvasObject{currentTab}

	// Обработчики для кнопок
	setActiveButton := func(activeBtn *widget.Button) {
		for _, btn := range buttons {
			btn.Importance = widget.MediumImportance
		}
		activeBtn.Importance = widget.HighImportance
	}

	setTab := func(tab fyne.CanvasObject, btn *widget.Button) {
		setActiveButton(btn)
		content.Objects = []fyne.CanvasObject{tab}
		content.Refresh()
	}

	cpuBtn.OnTapped = func() { setTab(tabs.CreateHardwareTab(window), cpuBtn) }
	appslibraryBtn.OnTapped = func() { setTab(tabs.CreateAppsLibraryTab(window), appslibraryBtn) }
	processBtn.OnTapped = func() { setTab(tabs.CreateProcessesTab(window), processBtn) }
	serverstatusBtn.OnTapped = func() { setTab(tabs.CreateServerStatusTab(window), serverstatusBtn) }
	settingsBtn.OnTapped = func() { setTab(tabs.CreateSettingsTab(window, myApp), serverstatusBtn) }

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
		currentVersion := "0.0.1" // Это должно быть из вашего приложения

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
