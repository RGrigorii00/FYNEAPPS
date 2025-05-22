package ui

import (
	"FYNEAPPS/database"
	"FYNEAPPS/resources"
	settings "FYNEAPPS/ui/setting_tab"
	"FYNEAPPS/ui/tabs"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows"
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
			widget.NewLabel("v0.0.17 alpha"),
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
		contentObj, _ := tabs.CreateTicketsTab(window)
		content.Objects = []fyne.CanvasObject{contentObj}
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

func updateApp(window fyne.Window) {
	progress := widget.NewProgressBarInfinite()
	statusLabel := widget.NewLabel(settings.GetLocalizedString("CheckingUpdates"))

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

		// 1. Получаем текущую версию приложения
		currentVersion := "0.0.17"

		// 2. Получаем информацию о последнем релизе
		statusLabel.SetText(settings.GetLocalizedString("FetchingReleaseInfo"))
		release, err := getLatestRelease("RGrigorii00", "FYNEAPPS")
		if err != nil {
			fmt.Print("Ошибка получения релиза", err)
			dialog.Hide()
			showErrorDialog(window, fmt.Sprintf("%s: %v", settings.GetLocalizedString("UpdateError"), err))
			return
		}

		// 3. Проверяем, нужно ли обновление
		if release.TagName == "v"+currentVersion {
			dialog.Hide()
			showInfoDialog(window,
				settings.GetLocalizedString("NoUpdateTitle"),
				fmt.Sprintf("%s\n\n%s: v%s\n%s: v%s",
					settings.GetLocalizedString("NoUpdateMessage"),
					settings.GetLocalizedString("CurrentVersion"),
					currentVersion,
					settings.GetLocalizedString("LatestVersion"),
					strings.TrimPrefix(release.TagName, "v"),
				))
			return
		}

		// 4. Ищем подходящий билд для текущей ОС
		statusLabel.SetText(settings.GetLocalizedString("FindingBuild"))
		assetURL := ""
		assetName := ""
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
			if strings.Contains(strings.ToLower(asset.Name), osPattern) {
				assetURL = asset.URL
				assetName = asset.Name
				break
			}
		}

		if assetURL == "" {
			dialog.Hide()
			showErrorDialog(window, fmt.Sprintf("%s\n\n%s: %s",
				settings.GetLocalizedString("NoBuildError"),
				settings.GetLocalizedString("YourOS"),
				runtime.GOOS))
			return
		}

		// 5. Скачиваем новый билд
		statusLabel.SetText(settings.GetLocalizedString("DownloadingUpdate"))
		exePath, err := os.Executable()
		if err != nil {
			dialog.Hide()
			showErrorDialog(window, fmt.Sprintf("%s\n\n%s: %v",
				settings.GetLocalizedString("ExePathError"),
				settings.GetLocalizedString("Details"),
				err))
			return
		}

		appDir := filepath.Dir(exePath)
		downloadPath := filepath.Join(appDir, assetName)

		// Скачиваем файл
		err = downloadFile(downloadPath, assetURL)
		if err != nil {
			dialog.Hide()
			showErrorDialog(window, fmt.Sprintf("%s\n\n%s: %v",
				settings.GetLocalizedString("DownloadError"),
				settings.GetLocalizedString("Details"),
				err))
			// Удаляем частично загруженный файл, если он существует
			if _, e := os.Stat(downloadPath); e == nil {
				os.Remove(downloadPath)
			}
			return
		}

		// 6. Подготовка к обновлению
		statusLabel.SetText(settings.GetLocalizedString("CleaningUp"))

		// Удаляем файл сессии
		sessionFile := filepath.Join(appDir, "session.json")
		if _, err := os.Stat(sessionFile); err == nil {
			if err := os.Remove(sessionFile); err != nil {
				fmt.Print("Ошибка удаления файла сессии", err)
			}
		}

		// Удаляем сессию из базы данных
		if sessionID := getCurrentSessionID(); sessionID != "" {
			if err := deleteCurrentSession(sessionID); err != nil {
				fmt.Print("Ошибка удаления сессии из БД", err)
			}
		}

		// 7. Устанавливаем обновление
		statusLabel.SetText(settings.GetLocalizedString("InstallingUpdate"))

		var installErr error
		if runtime.GOOS == "windows" {
			fmt.Print("Начато обновление (Windows)...")
			installErr = applyWindowsUpdate(exePath, downloadPath)
		} else {
			fmt.Print("Начато обновление (Unix)...")
			installErr = applyUnixUpdate(exePath, downloadPath)
		}

		if installErr != nil {
			dialog.Hide()
			showErrorDialog(window, fmt.Sprintf("%s\n\n%s: %v",
				settings.GetLocalizedString("InstallError"),
				settings.GetLocalizedString("Details"),
				installErr))
			// Удаляем загруженный файл обновления
			if _, e := os.Stat(downloadPath); e == nil {
				os.Remove(downloadPath)
			}
			return
		}

		// 8. Перезапускаем приложение
		restartApplication(exePath)
	}()
}

func applyWindowsUpdate(exePath, updatePath string) error {
	if runtime.GOOS == "windows" {
		// Получаем абсолютные пути
		exePath, _ = filepath.Abs(exePath)
		updatePath, _ = filepath.Abs(updatePath)

		// 1. Находим cmd.exe по абсолютному пути
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = "C:\\Windows"
		}
		cmdPath := filepath.Join(systemRoot, "System32", "cmd.exe")

		// 2. Создаем bat-файл с логикой обновления
		batContent := fmt.Sprintf(`@echo off
:: Административные права (если нужно)
:: if not "%1"=="admin" (powershell start -verb runas '%0' admin & exit /b)

:: Основной код обновления
echo [UPDATE] Завершаем текущий процесс...
taskkill /F /IM "%s" >nul 2>&1
ping -n 3 127.0.0.1 >nul

echo [UPDATE] Удаляем старую версию...
if exist "%s" (
    del /F /Q "%s" >nul 2>&1
    if exist "%s" (
        echo [ERROR] Не удалось удалить файл
        pause
        exit /B 1
    )
)

echo [UPDATE] Устанавливаем новую версию...
move /Y "%s" "%s" >nul 2>&1
if not exist "%s" (
    echo [ERROR] Не удалось переместить файл
    pause
    exit /B 2
)

echo [UPDATE] Запускаем обновленную версию...
start "" "%s"
exit /B 0
`, filepath.Base(exePath), exePath, exePath, exePath, updatePath, exePath, exePath, exePath)

		// 3. Сохраняем bat-файл рядом с exe
		batPath := filepath.Join(filepath.Dir(exePath), "update_"+filepath.Base(exePath)+".bat")
		if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
			return fmt.Errorf("ошибка создания bat-файла: %w", err)
		}

		// 4. Запускаем через абсолютный путь к cmd.exe
		if err := runCommandHidden(cmdPath, "/C", batPath); err != nil {
			return fmt.Errorf("ошибка запуска обновления: %w", err)
		}

		// 5. Немедленно завершаем текущее приложение
		os.Exit(0)
	}
	return nil
}

// runCommandHidden запускает команду без отображения окна (только для Windows)
func runCommandHidden(path string, args ...string) error {
	switch runtime.GOOS {
	case "windows":

		cmd := exec.Command(path, args...)
		cmd.SysProcAttr = &windows.SysProcAttr{
			HideWindow: true,
		}
		return cmd.Start()
	case "linux":
		// Просто запускаем команду без скрытия окна
		return exec.Command(path, args...).Start()
	default:

		// Просто запускаем команду без скрытия окна
		return exec.Command(path, args...).Start()
	}
	return nil
}

func applyUnixUpdate(exePath, updatePath string) error {
	// Устанавливаем права на новый файл
	if err := os.Chmod(updatePath, 0755); err != nil {
		return fmt.Errorf("ошибка установки прав: %w", err)
	}

	// Заменяем файл
	if err := os.Rename(updatePath, exePath); err != nil {
		return fmt.Errorf("ошибка замены файла: %w", err)
	}

	return nil
}

func restartApplication(exePath string) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Реализация для Windows
		cmd = exec.Command("cmd.exe", "/C", "start", "/B", `""`, exePath)
	} else {
		// Реализация для Linux и других Unix-систем
		cmd = exec.Command(exePath)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Print("Ошибка перезапуска приложения", err)
		return
	}

	// Даем процессу немного времени на запуск
	time.Sleep(500 * time.Millisecond)

	// Завершаем текущий процесс
	os.Exit(0)
}

// SessionData представляет структуру данных сессии
type SessionData struct {
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
	// Другие поля сессии, если они есть
}

func getCurrentSessionID() string {
	// Получаем путь к директории с исполняемым файлом
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	appDir := filepath.Dir(exePath)

	// Формируем путь к файлу session.json
	sessionFile := filepath.Join(appDir, "session.json")

	// Читаем файл
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return ""
	}

	// Парсим JSON
	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return ""
	}

	return session.SessionID
}

func deleteCurrentSession(sessionID string) error {
	connStr := "user=user dbname=grafana_db password=user host=83.166.245.249 port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return fmt.Errorf("ошибка проверки соединения: %w", err)
	}

	_, err = db.Exec("DELETE FROM user_sessions WHERE session_id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("ошибка удаления сессии: %w", err)
	}

	return nil
}

// Вспомогательные функции для отображения диалогов
func showErrorDialog(window fyne.Window, message string) {
	dialog := dialog.NewError(
		fmt.Errorf("ошибка обновления: %s", message),
		window,
	)
	dialog.Show()
}

func showInfoDialog(window fyne.Window, title, message string) {
	dialog := dialog.NewInformation(
		title,
		message,
		window,
	)
	dialog.Show()
}

// getLatestRelease получает информацию о последнем релизе
func getLatestRelease(owner, repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	// Создаем HTTP-клиент с таймаутом
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовки для GitHub API
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка HTTP-запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("неожиданный статус код: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
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
		fmt.Print("Ошибка при открытии браузера", err)
	}
}
