package tabs

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SystemSoftware struct {
	Name      string
	Version   string
	Publisher string
	Installed string
}

var (
	softwareCache   []SystemSoftware
	softwareCacheMu sync.Mutex
	lastUpdateTime  time.Time
	cacheExpiry     = 5 * time.Minute
)

func CreateSoftwareTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Программы на компьютере (Загружает дольше обычного)", theme.ForegroundColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Задаем фиксированные ширины столбцов
	columnWidths := []float32{
		710, // Название
		210, // Версия
		170, // Издатель
		150, // Дата установки
	}

	// Создаем заголовки таблицы
	headerRow := container.NewHBox()
	headers := []string{"Название", "Издатель", "Версия", "Дата установки"}

	// Указываем индивидуальные отступы для заголовков
	headerLeftPaddings := []float32{
		310, // Название
		370, // Версия
		110, // Издатель
		110, // Дата установки
	}

	for i, header := range headers {
		label := widget.NewLabel(header)
		label.TextStyle = fyne.TextStyle{Bold: true}

		paddedHeader := container.NewHBox()
		if i < len(headerLeftPaddings) {
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(headerLeftPaddings[i], 1))
			paddedHeader.Add(spacer)
		}
		paddedHeader.Add(label)
		headerRow.Add(paddedHeader)
	}

	// Создаем таблицу
	softwareTable := widget.NewTable(
		func() (int, int) { return 0, len(columnWidths) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignLeading
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			// Заполнение будет в updateSoftware
		},
	)

	// Устанавливаем ширины столбцов
	for i, width := range columnWidths {
		softwareTable.SetColumnWidth(i, width)
	}

	// Элементы управления (как во вкладке процессов)
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по названию или издателю...")
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)
	sortSelect := widget.NewSelect([]string{"Название", "Издатель", "Версия", "Дата установки"}, nil)
	sortSelect.SetSelected("Название")

	// Канал для остановки автообновления
	stopChan := make(chan struct{})

	// Функция получения данных с кэшированием
	getCachedSoftware := func() ([]SystemSoftware, error) {
		softwareCacheMu.Lock()
		defer softwareCacheMu.Unlock()

		if time.Since(lastUpdateTime) < cacheExpiry && len(softwareCache) > 0 {
			return softwareCache, nil
		}

		software, err := getInstalledSoftware()
		if err != nil {
			return nil, err
		}

		softwareCache = software
		lastUpdateTime = time.Now()
		return software, nil
	}

	// Функция обновления таблицы
	updateSoftware := func() {
		software, err := getCachedSoftware()
		if err != nil {
			log.Printf("Ошибка получения списка ПО: %v", err)
			return
		}

		filtered := filterSoftware(software, searchEntry.Text)
		sortSoftware(filtered, sortSelect.Selected)

		fyne.Do(func() {
			softwareTable.Length = func() (int, int) {
				return len(filtered), len(columnWidths)
			}
			softwareTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
				label := obj.(*widget.Label)
				if id.Row >= len(filtered) {
					label.SetText("")
					return
				}

				sw := filtered[id.Row]
				switch id.Col {
				case 0:
					label.SetText(sw.Name)
					label.Alignment = fyne.TextAlignLeading
				case 1:
					label.SetText(sw.Version)
					label.Alignment = fyne.TextAlignLeading
				case 2:
					label.SetText(sw.Publisher)
					label.Alignment = fyne.TextAlignLeading
				case 3:
					label.SetText(sw.Installed)
					label.Alignment = fyne.TextAlignLeading
				}
			}
			softwareTable.Refresh()
		})
	}

	// Автообновление (раз в 30 секунд)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				updateSoftware()
			case <-stopChan:
				return
			}
		}
	}()

	// Обработчики событий
	searchEntry.OnChanged = func(s string) { go updateSoftware() }
	refreshBtn.OnTapped = func() {
		softwareCacheMu.Lock()
		softwareCache = nil
		softwareCacheMu.Unlock()
		go updateSoftware()
	}
	sortSelect.OnChanged = func(s string) { go updateSoftware() }

	// Первоначальное обновление
	go updateSoftware()

	// Очистка при закрытии
	window.SetOnClosed(func() {
		close(stopChan)
	})

	// Компоновка элементов управления (как во вкладке процессов)
	controls := container.NewBorder(
		nil, nil, nil, nil,
		container.NewVBox(
			title,
			container.NewBorder(
				nil, nil,
				widget.NewLabel("Поиск:"),
				container.NewHBox(
					widget.NewLabel("Сортировка:"),
					sortSelect,
					refreshBtn,
				),
				searchEntry,
			),
		),
	)

	// Главный контейнер
	mainContent := container.NewBorder(
		controls,
		nil,
		nil,
		nil,
		container.NewBorder(
			headerRow,
			nil,
			nil,
			nil,
			container.NewScroll(softwareTable),
		),
	)

	return mainContent
}

func getInstalledSoftware() ([]SystemSoftware, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsSoftware()
	case "linux":
		return getLinuxSoftware()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func getWindowsSoftware() ([]SystemSoftware, error) {
	cmd := exec.Command("wmic", "product", "get", "name,version,vendor,installdate", "/format:csv")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("WMIC failed: %v", err)
	}

	lines := strings.Split(string(output), "\r\n")
	var softwareList []SystemSoftware

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 5 {
			installDate := parseWindowsInstallDate(parts[1])
			softwareList = append(softwareList, SystemSoftware{
				Name:      strings.TrimSpace(parts[2]),
				Version:   strings.TrimSpace(parts[3]),
				Publisher: strings.TrimSpace(parts[4]),
				Installed: installDate,
			})
		}
	}

	return softwareList, nil
}

func getLinuxSoftware() ([]SystemSoftware, error) {
	var softwareList []SystemSoftware

	// Получаем список пакетов через dpkg (Debian/Ubuntu)
	if _, err := os.Stat("/var/lib/dpkg/status"); err == nil {
		cmd := exec.Command("dpkg-query", "-W", "-f=${Package}\t${Version}\t${Maintainer}\t${Install-Date}\n")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("dpkg-query failed: %v", err)
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			parts := strings.Split(line, "\t")
			if len(parts) >= 4 {
				installDate := parseLinuxInstallDate(parts[3])
				softwareList = append(softwareList, SystemSoftware{
					Name:      strings.TrimSpace(parts[0]),
					Version:   strings.TrimSpace(parts[1]),
					Publisher: strings.TrimSpace(parts[2]),
					Installed: installDate,
				})
			}
		}
	}

	// Получаем список snap пакетов
	if _, err := exec.LookPath("snap"); err == nil {
		cmd := exec.Command("snap", "list")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("snap list failed: %v", err)
		} else {
			lines := strings.Split(string(output), "\n")
			for i, line := range lines {
				if i == 0 || strings.TrimSpace(line) == "" {
					continue
				}

				// Формат: Name Version Rev Tracking Publisher Notes
				fields := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), 6)
				if len(fields) >= 5 {
					softwareList = append(softwareList, SystemSoftware{
						Name:      fields[0],
						Version:   fields[1],
						Publisher: fields[4],
						Installed: "snap", // У snap нет даты установки в простом выводе
					})
				}
			}
		}
	}

	// Получаем список flatpak пакетов
	if _, err := exec.LookPath("flatpak"); err == nil {
		cmd := exec.Command("flatpak", "list", "--columns=application,version,origin,installation")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("flatpak list failed: %v", err)
		} else {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}

				fields := strings.Split(line, "\t")
				if len(fields) >= 3 {
					softwareList = append(softwareList, SystemSoftware{
						Name:      fields[0],
						Version:   fields[1],
						Publisher: fields[2],
						Installed: "flatpak", // Упрощенно
					})
				}
			}
		}
	}

	return softwareList, nil
}

func parseWindowsInstallDate(dateStr string) string {
	if len(dateStr) < 8 {
		return dateStr
	}
	return fmt.Sprintf("%s-%s-%s", dateStr[0:4], dateStr[4:6], dateStr[6:8])
}

func parseLinuxInstallDate(dateStr string) string {
	if len(dateStr) < 8 {
		return dateStr
	}
	// Формат даты в dpkg: 2023-04-20
	return dateStr
}

func filterSoftware(software []SystemSoftware, search string) []SystemSoftware {
	if search == "" {
		return software
	}

	var result []SystemSoftware
	search = strings.ToLower(search)
	for _, sw := range software {
		if strings.Contains(strings.ToLower(sw.Name), search) ||
			strings.Contains(strings.ToLower(sw.Publisher), search) {
			result = append(result, sw)
		}
	}
	return result
}

func sortSoftware(software []SystemSoftware, sortBy string) {
	switch sortBy {
	case "Название":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Name < software[j].Name
		})
	case "Издатель":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Version < software[j].Version
		})
	case "Версия":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Publisher < software[j].Publisher
		})
	case "Дата установки":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Installed > software[j].Installed
		})
	}
}
