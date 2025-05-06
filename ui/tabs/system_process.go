package tabs

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/process"
)

type ProcessInfo struct {
	PID     int32
	Name    string
	CPU     float64
	Memory  float32
	Status  string
	User    string
	Command string
}

func CreateProcessesTab(window fyne.Window) fyne.CanvasObject {
	// Создаем таблицу с заголовками
	tableWithHeaders := container.NewVBox()

	// Заголовки таблицы
	headers := container.NewGridWithColumns(6,
		widget.NewLabelWithStyle("PID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Имя процесса", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("CPU %", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Память (MB)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Статус", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Пользователь", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	tableWithHeaders.Add(headers)

	// Основная таблица процессов
	processTable := widget.NewTable(
		func() (int, int) { return 0, 6 },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			// Заполнение будет в updateProcesses
		},
	)

	// Настройка размеров столбцов
	// Используйте именованные поля:
	processTable.SetColumnWidth(0, 80)  // PID
	processTable.SetColumnWidth(1, 250) // Имя
	processTable.SetColumnWidth(2, 100) // CPU
	processTable.SetColumnWidth(3, 120) // Память
	processTable.SetColumnWidth(4, 100) // Статус
	processTable.SetColumnWidth(5, 200) // Увеличена ширина для пользователя

	tableWithHeaders.Add(container.NewStack(processTable))

	// Создаем контейнер с прокруткой
	log.Println("Создание контейнера с прокруткой...")
	scrollContainer := container.NewScroll(tableWithHeaders)
	log.Printf("Контейнер с прокруткой создан: %v", scrollContainer)

	// Устанавливаем минимальный размер
	minSize := fyne.NewSize(1000, 600)
	log.Printf("Установка минимального размера: %v", minSize)
	scrollContainer.SetMinSize(minSize)
	log.Println("Минимальный размер установлен")

	// Устанавливаем размер контейнера
	newSize := fyne.NewSize(1000, 1000)
	log.Printf("Установка размера контейнера: %v", newSize)
	scrollContainer.Resize(newSize)
	log.Println("Размер контейнера установлен")

	// Можно добавить проверку текущих размеров для отладки
	currentSize := scrollContainer.Size()
	log.Printf("Текущий размер контейнера: width=%.2f, height=%.2f", currentSize.Width, currentSize.Height)

	// Элементы управления
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по имени, PID или пользователю...")
	searchEntry.MinSize()
	searchEntry.Resize(fyne.NewSize(500, searchEntry.MinSize().Height)) // Увеличена ширина поля поиска

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)
	sortSelect := widget.NewSelect([]string{"CPU", "Память", "PID", "Имя"}, nil)
	sortSelect.SetSelected("CPU")

	// Канал для остановки автообновления
	stopChan := make(chan struct{})

	// Функция обновления списка процессов
	updateProcesses := func() {
		processes, err := getSystemProcesses()
		if err != nil {
			log.Printf("Ошибка получения процессов: %v", err)
			return
		}

		filtered := filterProcesses(processes, searchEntry.Text)
		sortProcesses(filtered, sortSelect.Selected)

		// Обновление в главном потоке
		fyne.Do(func() {
			processTable.Length = func() (int, int) {
				return len(filtered), 6
			}
			processTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
				label := obj.(*widget.Label)
				if id.Row >= len(filtered) {
					label.SetText("")
					return
				}

				proc := filtered[id.Row]
				switch id.Col {
				case 0:
					label.SetText(fmt.Sprintf("%d", proc.PID))
				case 1:
					label.SetText(proc.Name)
				case 2:
					label.SetText(fmt.Sprintf("%.1f%%", proc.CPU))
				case 3:
					label.SetText(fmt.Sprintf("%.1f", proc.Memory))
				case 4:
					// Обработка пустого статуса
					if proc.Status == "" {
						label.SetText("-")
					} else {
						label.SetText(proc.Status)
					}
				case 5:
					// Обрезаем длинные имена пользователей
					if len(proc.User) > 20 {
						label.SetText(proc.User[:20] + "...")
					} else {
						label.SetText(proc.User)
					}
				}
			}
			processTable.Refresh()
		})
	}

	// Оптимизированное автообновление
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				updateProcesses()
			case <-stopChan:
				return
			}
		}
	}()

	// Обработчики событий
	searchEntry.OnChanged = func(s string) { go updateProcesses() }
	refreshBtn.OnTapped = func() { go updateProcesses() }
	sortSelect.OnChanged = func(s string) { go updateProcesses() }

	// Первоначальное обновление в фоне
	go updateProcesses()

	// Очистка при закрытии
	window.SetOnClosed(func() {
		close(stopChan)
	})

	// Компоновка интерфейса
	searchContainer := container.NewVBox(
		container.NewBorder(nil, nil, nil, nil, searchEntry),
		layout.NewSpacer(),
		widget.NewLabel("Сортировка:"),
		sortSelect,
		refreshBtn,
	)

	// Главный контейнер с растягиванием
	mainContent := container.NewBorder(
		searchContainer,
		nil,
		nil,
		nil,
		scrollContainer,
	)

	return mainContent
}

func getSystemProcesses() ([]ProcessInfo, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var result []ProcessInfo
	for _, p := range processes {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status() // Теперь это string, а не []string
		user, _ := p.Username()
		cmd, _ := p.Cmdline()

		result = append(result, ProcessInfo{
			PID:     p.Pid,
			Name:    name,
			CPU:     cpu,
			Memory:  float32(mem),
			Status:  status, // Просто используем string напрямую
			User:    user,
			Command: cmd,
		})
	}

	return result, nil
}

func filterProcesses(processes []ProcessInfo, search string) []ProcessInfo {
	if search == "" {
		return processes
	}

	var result []ProcessInfo
	for _, p := range processes {
		if containsProcessInfo(p, search) {
			result = append(result, p)
		}
	}
	return result
}

func containsProcessInfo(p ProcessInfo, search string) bool {
	search = strings.ToLower(search)
	return strings.Contains(strings.ToLower(p.Name), search) ||
		strings.Contains(strconv.Itoa(int(p.PID)), search) ||
		strings.Contains(strings.ToLower(p.User), search) ||
		strings.Contains(strings.ToLower(p.Command), search)
}

func sortProcesses(processes []ProcessInfo, sortBy string) {
	switch sortBy {
	case "CPU":
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].CPU > processes[j].CPU
		})
	case "Память":
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].Memory > processes[j].Memory
		})
	case "PID":
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].PID < processes[j].PID
		})
	case "Имя":
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].Name < processes[j].Name
		})
	}
}
