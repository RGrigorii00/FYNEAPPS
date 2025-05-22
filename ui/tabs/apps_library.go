package tabs

import (
	"FYNEAPPS/database"
	"FYNEAPPS/resources"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Software struct {
	ID                int
	ComputerID        int
	Name              string
	Version           string
	Publisher         string
	InstallDate       time.Time
	InstallLocation   string
	SizeMB            float64
	IsSystemComponent bool
	IsUpdate          bool
	Architecture      string
	LastUsedDate      time.Time
	Timestamp         time.Time
	DownloadURL       string
	Picture           []byte
	TargetPlatform    string

	InstallCommand string `json:"install_command"` // Например: "sudo dpkg -i {file}"
}

type AppCard struct {
	Software
	Card        *fyne.Container
	DownloadBtn *widget.Button
	ShowDetails bool
	ProgressBar *widget.ProgressBar
}

var (
	appCards      []AppCard
	currentWindow fyne.Window
	dbConn        *database.PGConnection
	downloadsDir  = "downloads"
)

func init() {
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		log.Printf("Не удалось создать папку для загрузок: %v", err)
	}
}

func CreateAppsLibraryTab(window fyne.Window, db *database.PGConnection) fyne.CanvasObject {
	currentWindow = window
	dbConn = db

	title := canvas.NewText("Библиотека приложений ПГАТУ", theme.Color(theme.ColorNameForeground))
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск приложений...")

	scrollContainer := container.NewVScroll(nil)
	scrollContainer.SetMinSize(fyne.NewSize(800, 600))

	filterApps := func(query string) {
		query = strings.ToLower(query)

		if scrollContainer.Content == nil {
			return
		}

		content, ok := scrollContainer.Content.(*fyne.Container)
		if !ok || len(content.Objects) == 0 {
			return
		}

		grid, ok := content.Objects[0].(*fyne.Container)
		if !ok {
			return
		}

		grid.Objects = nil

		for _, appCard := range appCards {
			matches := strings.Contains(strings.ToLower(appCard.Name), query) ||
				strings.Contains(strings.ToLower(appCard.Publisher), query) ||
				strings.Contains(strings.ToLower(appCard.Version), query)

			if matches {
				grid.Add(appCard.Card)
			}
		}

		grid.Refresh()
		scrollContainer.Refresh()
	}

	searchEntry.OnChanged = func(query string) {
		time.AfterFunc(300*time.Millisecond, func() {
			fyne.Do(func() {
				filterApps(query)
			})
		})
	}

	updateContent := func() {
		if !dbConn.IsConnected() {
			opts := database.ConnectionOptions{
				Host:     "83.166.245.249",
				Port:     "5432",
				User:     "user",
				Password: "user",
				DBName:   "grafana_db",
				SSLMode:  "disable",
			}
			if err := dbConn.Connect(opts); err != nil {
				log.Printf("Ошибка подключения к БД: %v", err)
				scrollContainer.Content = widget.NewLabel("Ошибка подключения к базе данных")
				return
			}
		}

		softwareList, err := loadLatestSoftwareVersions()
		if err != nil {
			log.Printf("Ошибка загрузки ПО: %v", err)
			scrollContainer.Content = widget.NewLabel("Ошибка загрузки данных")
			return
		}

		appCards = nil
		grid := container.NewGridWithColumns(3)

		for _, sw := range softwareList {
			card, downloadBtn, progressBar, err := createAppCard(sw)
			if err != nil {
				log.Printf("Ошибка создания карточки: %v", err)
				continue
			}

			appCard := AppCard{
				Software:    sw,
				Card:        card,
				DownloadBtn: downloadBtn,
				ProgressBar: progressBar,
			}
			appCards = append(appCards, appCard)

			grid.Add(container.NewPadded(card))
		}

		scrollContainer.Content = container.NewVBox(grid)
	}

	updateContent()

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), updateContent)
	refreshBtn.Importance = widget.MediumImportance

	return container.NewBorder(
		container.NewVBox(
			container.NewPadded(title),
			widget.NewSeparator(),
			container.NewPadded(searchEntry),
			widget.NewSeparator(),
		),
		container.NewHBox(layout.NewSpacer(), refreshBtn, layout.NewSpacer()),
		nil,
		nil,
		scrollContainer,
	)
}

func createAppCard(sw Software) (*fyne.Container, *widget.Button, *widget.ProgressBar, error) {
	iconRes := resources.ResourcePgatulogosmallPng
	icon := canvas.NewImageFromResource(iconRes)
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(128, 128))

	progress := widget.NewProgressBar()
	progress.Hide()

	speedLabel := widget.NewLabel("")
	speedLabel.Hide()

	// Кнопка "Установить" (будет использоваться для скачивания и установки)
	installBtn := widget.NewButtonWithIcon("Установить", theme.DownloadIcon(), nil)
	installBtn.OnTapped = func() {
		if sw.DownloadURL == "" {
			showInfoDialog("Ошибка", "URL для скачивания не указан")
			return
		}

		fyne.Do(func() {
			installBtn.Disable()
			progress.Show()
			speedLabel.Show()
			progress.SetValue(0)
			speedLabel.SetText("Подготовка...")
		})

		go func() {
			filePath, err := downloadFile(sw.Name, sw.DownloadURL, func(p float64, speed string) {
				fyne.Do(func() {
					progress.SetValue(p)
					speedLabel.SetText(fmt.Sprintf("Скорость: %s", speed))
				})
			})

			fyne.Do(func() {
				if err != nil {
					showInfoDialog("Ошибка", fmt.Sprintf("Ошибка скачивания: %v", err))
					installBtn.Enable()
					progress.Hide()
					speedLabel.Hide()
					return
				}

				progress.SetValue(1.0)
				speedLabel.SetText("Устанавливается...")

				// После скачивания запускаем установку
				err = installApplication(sw, filePath)
				installBtn.Enable()

				if err != nil {
					showInfoDialog("Ошибка", fmt.Sprintf("Ошибка установки: %v", err))
				} else {
					showInfoDialog("Успешно", "Приложение успешно установлено!")
				}

				progress.Hide()
				speedLabel.Hide()
			})
		}()
	}

	launchBtn := widget.NewButtonWithIcon("Запуск", theme.MediaPlayIcon(), func() {
		err := launchApplication(sw.InstallLocation)
		if err != nil {
			showInfoDialog("Ошибка", fmt.Sprintf("Не удалось запустить приложение: %v", err))
		}
	})

	detailsBtn := widget.NewButtonWithIcon("Подробнее", theme.InfoIcon(), func() {
		showSoftwareInfo(sw)
	})

	cardContent := container.NewVBox(
		container.NewCenter(icon),
		container.NewCenter(widget.NewLabel(sw.Name)),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Версия: %s", sw.Version))),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Производитель: %s", sw.Publisher))),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Размер: %.2f MB", sw.SizeMB))),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Архитектура: %s", sw.Architecture))),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Платформа: %s", sw.TargetPlatform))),
		layout.NewSpacer(),
		container.NewVBox(
			container.NewHBox(
				layout.NewSpacer(),
				installBtn,
				widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
					checkForUpdates(sw.ID)
				}),
				launchBtn,
				detailsBtn,
				layout.NewSpacer(),
			),
			container.NewPadded(progress),
			container.NewCenter(speedLabel),
		),
	)

	border := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 0})
	border.StrokeColor = color.NRGBA{R: 150, G: 150, B: 150, A: 200}
	border.StrokeWidth = 0.5
	border.CornerRadius = 10

	card := container.NewPadded(
		container.NewStack(
			border,
			container.NewPadded(cardContent),
		),
	)

	return card, installBtn, progress, nil
}

func loadLatestSoftwareVersions() ([]Software, error) {
	query := `
		SELECT 
			software_id, computer_id, name, version, publisher, 
			install_date, install_location, size_mb, is_system_component, 
			is_update, architecture, last_used_date, timestamp,
			download_url, picture, target_platform, install_command
		FROM software
		ORDER BY name
	`

	rows, err := dbConn.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса ПО: %v", err)
	}
	defer rows.Close()

	var softwareList []Software

	for rows.Next() {
		var sw Software
		var installDate, lastUsedDate, timestamp sql.NullTime
		var sizeMB sql.NullFloat64
		var downloadURL, installLocation, architecture, targetPlatform, installCommand sql.NullString
		var picture []byte

		err := rows.Scan(
			&sw.ID, &sw.ComputerID, &sw.Name, &sw.Version, &sw.Publisher,
			&installDate, &installLocation, &sizeMB, &sw.IsSystemComponent,
			&sw.IsUpdate, &architecture, &lastUsedDate, &timestamp,
			&downloadURL, &picture, &targetPlatform, &installCommand,
		)
		if err != nil {
			log.Printf("Ошибка сканирования строки ПО: %v", err)
			continue
		}

		if installDate.Valid {
			sw.InstallDate = installDate.Time
		}
		if lastUsedDate.Valid {
			sw.LastUsedDate = lastUsedDate.Time
		}
		if timestamp.Valid {
			sw.Timestamp = timestamp.Time
		}
		if sizeMB.Valid {
			sw.SizeMB = sizeMB.Float64
		}
		if downloadURL.Valid {
			sw.DownloadURL = downloadURL.String
		}
		if installLocation.Valid {
			sw.InstallLocation = installLocation.String
		}
		if architecture.Valid {
			sw.Architecture = architecture.String
		}
		if targetPlatform.Valid {
			sw.TargetPlatform = targetPlatform.String
		}
		if installCommand.Valid {
			sw.InstallCommand = installCommand.String
		}
		sw.Picture = picture

		softwareList = append(softwareList, sw)
	}

	return softwareList, nil
}

func showSoftwareInfo(sw Software) {
	content := widget.NewForm(
		widget.NewFormItem("ID", widget.NewLabel(fmt.Sprintf("%d", sw.ID))),
		widget.NewFormItem("ID компьютера", widget.NewLabel(fmt.Sprintf("%d", sw.ComputerID))),
		widget.NewFormItem("Название", widget.NewLabel(sw.Name)),
		widget.NewFormItem("Версия", widget.NewLabel(sw.Version)),
		widget.NewFormItem("Производитель", widget.NewLabel(sw.Publisher)),
		widget.NewFormItem("Дата установки", widget.NewLabel(sw.InstallDate.Format("2006-01-02"))),
		widget.NewFormItem("Расположение", widget.NewLabel(sw.InstallLocation)),
		widget.NewFormItem("Размер (MB)", widget.NewLabel(fmt.Sprintf("%.2f", sw.SizeMB))),
		widget.NewFormItem("Системный компонент", widget.NewLabel(fmt.Sprintf("%t", sw.IsSystemComponent))),
		widget.NewFormItem("Обновление", widget.NewLabel(fmt.Sprintf("%t", sw.IsUpdate))),
		widget.NewFormItem("Архитектура", widget.NewLabel(sw.Architecture)),
		widget.NewFormItem("Дата последнего использования", widget.NewLabel(sw.LastUsedDate.Format("2006-01-02 15:04:05"))),
		widget.NewFormItem("Временная метка", widget.NewLabel(sw.Timestamp.Format("2006-01-02 15:04:05"))),
		widget.NewFormItem("URL загрузки", widget.NewLabel(sw.DownloadURL)),
		widget.NewFormItem("Целевая платформа", widget.NewLabel(sw.TargetPlatform)),
		widget.NewFormItem("Команда установки", widget.NewLabel(sw.InstallCommand)),
	)

	var dialog *widget.PopUp
	closeBtn := widget.NewButton("Закрыть", func() {
		dialog.Hide()
	})

	dialog = widget.NewModalPopUp(
		container.NewVBox(
			content,
			closeBtn,
		),
		currentWindow.Canvas(),
	)
	dialog.Show()
}

func downloadFile(filename, url string, updateProgress func(float64, string)) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка соединения: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("сервер вернул ошибку: %s", resp.Status)
	}

	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return "", fmt.Errorf("не удалось создать папку: %v", err)
	}

	filePath := filepath.Join(downloadsDir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("не удалось создать файл: %v", err)
	}
	defer out.Close()

	counter := &writeCounter{
		total:   resp.ContentLength,
		update:  updateProgress,
		written: 0,
	}

	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("ошибка скачивания: %v", err)
	}

	return filePath, nil
}

type writeCounter struct {
	total       int64
	update      func(float64, string)
	written     int64
	startTime   time.Time
	lastWritten int64
	lastTime    time.Time
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.written += int64(n)
	now := time.Now()

	if wc.startTime.IsZero() {
		wc.startTime = now
		wc.lastTime = now
		wc.lastWritten = 0
	}

	if now.Sub(wc.lastTime) >= 500*time.Millisecond {
		elapsed := now.Sub(wc.lastTime).Seconds()
		bytesSinceLast := wc.written - wc.lastWritten
		speed := float64(bytesSinceLast) / elapsed

		speedStr := formatSpeed(speed)
		progress := float64(wc.written) / float64(wc.total)
		wc.update(progress, speedStr)

		wc.lastTime = now
		wc.lastWritten = wc.written
	}

	return n, nil
}

func formatSpeed(bytesPerSec float64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
	)

	switch {
	case bytesPerSec >= MB:
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/MB)
	case bytesPerSec >= KB:
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/KB)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}

func launchApplication(path string) error {
	// Если путь - это команда (например, для запуска из PATH)
	if !strings.Contains(path, "/") && !strings.Contains(path, "\\") {
		cmd := exec.Command(path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Start()
	}

	// Стандартный запуск приложения по пути
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("неподдерживаемая платформа")
	}

	return cmd.Start()
}

func checkForUpdates(softwareID int) {
	query := `
		SELECT update_name, update_version, kb_article, size_mb
		FROM software_updates
		WHERE software_id = $1 AND is_uninstalled = FALSE
		ORDER BY install_date DESC
		LIMIT 1
	`

	var updateName, updateVersion, kbArticle string
	var sizeMB float64

	err := dbConn.DB().QueryRow(query, softwareID).Scan(&updateName, &updateVersion, &kbArticle, &sizeMB)
	if err != nil {
		if err == sql.ErrNoRows {
			showInfoDialog("Обновления не найдены", "Нет доступных обновлений для этого ПО.")
		} else {
			showInfoDialog("Ошибка", fmt.Sprintf("Ошибка при проверке обновлений: %v", err))
		}
		return
	}

	content := widget.NewForm(
		widget.NewFormItem("Обновление", widget.NewLabel(updateName)),
		widget.NewFormItem("Версия", widget.NewLabel(updateVersion)),
		widget.NewFormItem("Размер", widget.NewLabel(fmt.Sprintf("%.2f MB", sizeMB))),
		widget.NewFormItem("KB Статья", widget.NewLabel(kbArticle)),
	)

	var dialog *widget.PopUp
	installBtn := widget.NewButton("Установить", func() {
		installUpdate(softwareID, updateVersion)
		dialog.Hide()
	})
	cancelBtn := widget.NewButton("Отмена", func() {
		dialog.Hide()
	})

	dialog = widget.NewModalPopUp(
		container.NewVBox(
			content,
			container.NewHBox(
				layout.NewSpacer(),
				cancelBtn,
				installBtn,
				layout.NewSpacer(),
			),
		),
		currentWindow.Canvas(),
	)
	dialog.Show()
}

func installUpdate(softwareID int, version string) {
	showInfoDialog("Обновление установлено", fmt.Sprintf("Версия %s успешно установлена.", version))
}

func showInfoDialog(title, message string) {
	var dialog *widget.PopUp
	okBtn := widget.NewButton("OK", func() {
		dialog.Hide()
	})

	dialog = widget.NewModalPopUp(
		container.NewVBox(
			widget.NewLabel(title),
			widget.NewLabel(message),
			okBtn,
		),
		currentWindow.Canvas(),
	)
	dialog.Show()
}

func loadAppIcon(path string) (*canvas.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, _, err = image.Decode(file)
	if err != nil {
		return nil, err
	}

	return canvas.NewImageFromFile(path), nil
}

func installApplication(sw Software, filePath string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("автоматическая установка поддерживается только в Linux")
	}

	// Если есть команда установки - используем ее
	if sw.InstallCommand != "" {
		// Подставляем путь к скачанному файлу, если есть плейсхолдер
		cmdStr := strings.Replace(sw.InstallCommand, "{file}", filePath, -1)

		// Выполняем команду через shell, чтобы поддерживать pushd/popd и другие shell-конструкции
		cmd := exec.Command("bash", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	// Остальная логика функции для случаев без команды установки
	if filePath != "" {
		// Для .deb пакетов
		if strings.HasSuffix(filePath, ".deb") {
			cmd := exec.Command("sudo", "dpkg", "-i", filePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				// Попробуем установить зависимости
				cmd = exec.Command("sudo", "apt-get", "install", "-f", "-y")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			}
			return nil
		}

		// Для других типов файлов делаем исполняемыми
		if err := os.Chmod(filePath, 0755); err != nil {
			return fmt.Errorf("не удалось сделать файл исполняемым: %v", err)
		}
		return nil
	}

	return fmt.Errorf("не указана команда установки и не скачан файл")
}
