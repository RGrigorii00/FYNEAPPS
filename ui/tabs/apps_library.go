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
}

type AppCard struct {
	Software
	Card        *fyne.Container
	DownloadBtn *widget.Button
	ShowDetails bool // Добавляем поле для отслеживания состояния
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

	// Создаем поле поиска
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск приложений...")

	searchEntry.OnChanged = func(query string) {
		// Добавляем debounce на 300ms
		time.AfterFunc(300*time.Millisecond, func() {
			fyne.Do(func() {
				filterApps(query)
			})
		})
	}

	scrollContainer := container.NewVScroll(nil)
	scrollContainer.SetMinSize(fyne.NewSize(800, 600))

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
		grid.Layout = layout.NewAdaptiveGridLayout(3) // Используем адаптивный layout

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

		for i := len(softwareList); i%3 != 0; i++ {
			grid.Add(container.NewPadded(widget.NewLabel("")))
		}

		scrollContainer.Content = container.NewVBox(
			// container.NewPadded(title),
			// widget.NewSeparator(),
			grid,
		)
	}

	updateContent()

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), updateContent)
	refreshBtn.Importance = widget.MediumImportance

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			updateContent()
		}
	}()

	return container.NewBorder(
		container.NewVBox(
			container.NewPadded(title),
			widget.NewSeparator(),
			container.NewPadded(searchEntry), // Добавляем поле поиска
			widget.NewSeparator(),
		),
		container.NewHBox(layout.NewSpacer(), refreshBtn, layout.NewSpacer()),
		nil,
		nil,
		scrollContainer,
	)
}

func filterApps(query string) {
	query = strings.ToLower(query)

	// Сначала скроем все карточки, которые не соответствуют фильтру
	for _, appCard := range appCards {
		matches := strings.Contains(strings.ToLower(appCard.Name), query) ||
			strings.Contains(strings.ToLower(appCard.Publisher), query) ||
			strings.Contains(strings.ToLower(appCard.Version), query)

		appCard.Card.Hidden = !matches
	}

	// Затем обновим layout, чтобы оставшиеся карточки сместились наверх
	if currentWindow != nil {
		currentWindow.Content().Refresh()
	}
}

func createAppCard(sw Software) (*fyne.Container, *widget.Button, *widget.ProgressBar, error) {
	// Используем встроенный ресурс вместо загрузки из файла
	iconRes := resources.ResourcePgatulogosmallPng // Автоматически сгенерированное имя

	// Создаем изображение из ресурса
	icon := canvas.NewImageFromResource(iconRes)
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(128, 128))

	// Остальной код остается без изменений...
	progress := widget.NewProgressBar()
	progress.Hide()

	speedLabel := widget.NewLabel("")
	speedLabel.Hide()

	downloadBtn := widget.NewButtonWithIcon("Скачать", theme.DownloadIcon(), nil)
	downloadBtn.OnTapped = func() {
		if sw.DownloadURL == "" {
			showInfoDialog("Ошибка", "URL для скачивания не указан")
			return
		}

		fyne.Do(func() {
			downloadBtn.Disable()
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
				downloadBtn.Enable()
				if err != nil {
					showInfoDialog("Ошибка", fmt.Sprintf("Ошибка скачивания: %v", err))
					progress.Hide()
					speedLabel.Hide()
					return
				}

				progress.SetValue(1.0)
				speedLabel.SetText("Завершено")
				showInfoDialog("Успешно", fmt.Sprintf("Файл сохранен в: %s", filePath))
				time.AfterFunc(2*time.Second, func() {
					fyne.Do(func() {
						progress.Hide()
						speedLabel.Hide()
					})
				})
			})
		}()
	}

	// Создаем лейбл для деталей (изначально скрыт)
	detailsLabel := widget.NewLabel("")
	detailsLabel.Wrapping = fyne.TextWrapWord
	detailsLabel.Hide()

	// Функция для обновления текста деталей
	updateDetails := func() {
		detailsText := fmt.Sprintf(
			"Дата установки: %s\n"+
				"Расположение: %s\n"+
				"Размер: %.2f MB\n"+
				"Архитектура: %s\n"+
				"Производитель: %s",
			sw.InstallDate.Format("2006-01-02"),
			sw.InstallLocation,
			sw.SizeMB,
			sw.Architecture,
			sw.Publisher,
		)
		detailsLabel.SetText(detailsText)
	}

	// Создаем кнопку "Подробнее"
	detailsBtn := widget.NewButtonWithIcon("Подробнее", theme.InfoIcon(), func() {
		// Переключаем состояние
		showSoftwareInfo(sw)
		for i := range appCards {
			if appCards[i].ID == sw.ID {
				appCards[i].ShowDetails = !appCards[i].ShowDetails
				break
			}
		}

		// Обновляем отображение
		if detailsLabel.Visible() {
			detailsLabel.Hide()
		} else {
			updateDetails()
			detailsLabel.Show()
		}

		// Обновляем интерфейс
		currentWindow.Content().Refresh()
	})

	// Создаем карточку с рамкой и центрированным содержимым
	cardContent := container.NewVBox(
		container.NewCenter(icon),
		container.NewCenter(widget.NewLabel(sw.Name)),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Версия: %s", sw.Version))),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Производитель: %s", sw.Publisher))),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Размер: %.2f MB", sw.SizeMB))),
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Архитектура: %s", sw.Architecture))),
		layout.NewSpacer(),
		container.NewVBox(
			container.NewHBox(
				layout.NewSpacer(),
				downloadBtn,
				widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
					checkForUpdates(sw.ID)
				}),
				widget.NewButtonWithIcon("Запуск", theme.MediaPlayIcon(), func() {
					if sw.InstallLocation != "" {
						err := launchApplication(sw.InstallLocation)
						if err != nil {
							showInfoDialog("Ошибка", fmt.Sprintf("Не удалось запустить приложение: %v", err))
						}
					} else {
						showInfoDialog("Ошибка", "Путь к приложению не указан в базе данных")
					}
				}),

				detailsBtn, // Новая кнопка
				layout.NewSpacer(),
			),
			container.NewPadded( // Контейнер с отступами для прогресс-бара
				progress, // Теперь будет занимать всю доступную ширину
			),
			container.NewCenter(speedLabel),
		),
	)

	// Создаем скругленную рамку вокруг карточки
	border := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 0}) // Прозрачная заливка
	border.StrokeColor = color.NRGBA{R: 150, G: 150, B: 150, A: 200}         // Цвет рамки (серый)
	border.StrokeWidth = 0.5                                                 // Тонкая линия
	border.CornerRadius = 10                                                 // Радиус скругления углов (в пикселях)

	card := container.NewPadded(
		container.NewStack(
			border,
			container.NewPadded(
				cardContent,
			),
		),
	)

	return card, downloadBtn, progress, nil
}

func loadLatestSoftwareVersions() ([]Software, error) {
	query := `
		SELECT s.software_id, s.name, s.version, s.publisher, s.install_date, 
		       s.install_location, s.size_mb, s.is_system_component, 
		       s.is_update, s.architecture, s.last_used_date, s.timestamp,
		       s.download_url
		FROM software s
		INNER JOIN (
			SELECT name, MAX(timestamp) as latest_timestamp
			FROM software
			GROUP BY name
		) latest ON s.name = latest.name AND s.timestamp = latest.latest_timestamp
		--WHERE s.is_system_component = FALSE
		ORDER BY s.name
	`

	rows, err := dbConn.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса ПО: %v", err)
	}
	defer rows.Close()

	var softwareList []Software

	for rows.Next() {
		var sw Software
		err := rows.Scan(
			&sw.ID, &sw.Name, &sw.Version, &sw.Publisher, &sw.InstallDate,
			&sw.InstallLocation, &sw.SizeMB, &sw.IsSystemComponent,
			&sw.IsUpdate, &sw.Architecture, &sw.LastUsedDate, &sw.Timestamp,
			&sw.DownloadURL,
		)
		if err != nil {
			log.Printf("Ошибка сканирования строки ПО: %v", err)
			continue
		}
		softwareList = append(softwareList, sw)
	}

	return softwareList, nil
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

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("ошибка запуска: %v", err)
	}

	return nil
}

func showSoftwareInfo(sw Software) {
	content := widget.NewForm(
		widget.NewFormItem("Название", widget.NewLabel(sw.Name)),
		widget.NewFormItem("Версия", widget.NewLabel(sw.Version)),
		widget.NewFormItem("Производитель", widget.NewLabel(sw.Publisher)),
		widget.NewFormItem("Дата установки", widget.NewLabel(sw.InstallDate.Format("2006-01-02"))),
		widget.NewFormItem("Расположение", widget.NewLabel(sw.InstallLocation)),
		widget.NewFormItem("Размер", widget.NewLabel(fmt.Sprintf("%.2f MB", sw.SizeMB))),
		widget.NewFormItem("Архитектура", widget.NewLabel(sw.Architecture)),
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
