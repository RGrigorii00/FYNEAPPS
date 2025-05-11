package tabs

import (
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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
	title := canvas.NewText("Программы на компьютере", theme.Color(theme.ColorNameForeground))
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}
	// Основная таблица ПО
	softwareTable := widget.NewTable(
		func() (int, int) { return 0, 4 },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextTruncate
		},
	)

	// Настройка размеров столбцов
	softwareTable.SetColumnWidth(0, 500) // Название (самый широкий)
	softwareTable.SetColumnWidth(1, 300) // Версия
	softwareTable.SetColumnWidth(2, 300) // Издатель
	softwareTable.SetColumnWidth(3, 150) // Дата установки

	// Контейнер с заголовками и таблицей
	tableContainer := container.NewBorder(
		container.NewGridWithColumns(4,
			widget.NewLabelWithStyle("Название", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Версия", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Издатель", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Дата установки", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		nil, nil, nil,
		container.NewScroll(softwareTable),
	)

	// Элементы управления с фиксированной минимальной шириной
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по названию или издателю...")
	//searchEntry.MinSize().Width = 300 // Фиксированная минимальная ширина поля поиска

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)
	sortSelect := widget.NewSelect([]string{"Название", "Версия", "Издатель", "Дата установки"}, nil)
	sortSelect.SetSelected("Название")
	//sortSelect.MinSize().Width = 150 // Фиксированная минимальная ширина выпадающего списка

	// Канал для остановки автообновления
	stopChan := make(chan struct{})

	// Функция получения данных с кэшированием
	getCachedSoftware := func() ([]SystemSoftware, error) {
		softwareCacheMu.Lock()
		defer softwareCacheMu.Unlock()

		if time.Since(lastUpdateTime) < cacheExpiry && len(softwareCache) > 0 {
			return softwareCache, nil
		}

		software, err := getWindowsSoftware()
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
				return len(filtered), 4
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
					label.Wrapping = fyne.TextWrapBreak
				case 1:
					label.SetText(sw.Version)
					label.Wrapping = fyne.TextTruncate
				case 2:
					label.SetText(sw.Publisher)
					label.Wrapping = fyne.TextWrapBreak
				case 3:
					label.SetText(sw.Installed)
					label.Wrapping = fyne.TextTruncate
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

	// Компоновка элементов управления с правильными пропорциями
	searchContainer := container.NewHBox(
		widget.NewLabel("Поиск:"),
		searchEntry,
	)
	searchContainer.Layout = layout.NewBorderLayout(nil, nil, nil, nil)

	sortContainer := container.NewHBox(
		widget.NewLabel("Сортировка:"),
		sortSelect,
		refreshBtn,
	)

	controls := container.NewBorder(
		title,
		nil, nil,
		searchContainer,
		sortContainer,
	)

	// Главный контейнер
	mainContent := container.NewBorder(
		controls,
		nil,
		nil,
		nil,
		tableContainer,
	)

	return mainContent
}

// Остальные функции без изменений
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
		fmt.Print(parts)
		if len(parts) >= 5 {
			installDate := parseWindowsInstallDate(parts[1])
			softwareList = append(softwareList, SystemSoftware{
				Name:      strings.TrimSpace(parts[2]),
				Version:   strings.TrimSpace(parts[3]),
				Publisher: strings.TrimSpace(parts[4]),
				Installed: installDate,
			})
			fmt.Print(softwareList)
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
	case "Версия":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Version < software[j].Version
		})
	case "Издатель":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Publisher < software[j].Publisher
		})
	case "Дата установки":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Installed > software[j].Installed
		})
	}
}
