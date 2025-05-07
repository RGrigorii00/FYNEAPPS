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

	// Настройка размеров столбцов (автоматическое растягивание)
	processTable.SetColumnWidth(0, 100) // PID
	processTable.SetColumnWidth(1, 300) // Имя
	processTable.SetColumnWidth(2, 100) // CPU
	processTable.SetColumnWidth(3, 150) // Память
	processTable.SetColumnWidth(4, 150) // Статус
	processTable.SetColumnWidth(5, 200) // Пользователь

	// Создаем контейнер с заголовками и таблицей
	tableContainer := container.NewBorder(
		// Заголовки таблицы
		container.NewGridWithColumns(6,
			widget.NewLabelWithStyle("PID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Имя процесса", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("CPU %", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Память (MB)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Статус", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Пользователь", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		nil, nil, nil,
		container.NewScroll(processTable),
	)

	// Элементы управления
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по имени, PID или пользователю...")
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
					label.SetText(proc.Status)
				case 5:
					label.SetText(proc.User) // Убрано сокращение пользователя
				}
			}
			processTable.Refresh()
		})
	}

	// Автообновление
	go func() {
		ticker := time.NewTicker(1 * time.Second)
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

	// Первоначальное обновление
	go updateProcesses()

	// Очистка при закрытии
	window.SetOnClosed(func() {
		close(stopChan)
	})

	// Компоновка элементов управления
	controls := container.NewHBox(
		container.NewVBox(
			container.NewHBox(
				widget.NewLabel("Поиск:"),
				searchEntry,
			),
		),
		layout.NewSpacer(),
		container.NewHBox(
			widget.NewLabel("Сортировка:"),
			sortSelect,
			refreshBtn,
		),
	)

	// Главный контейнер с растягиванием на весь экран
	mainContent := container.NewBorder(
		controls,
		nil,
		nil,
		nil,
		container.NewMax(tableContainer), // Используем Max для растягивания
	)

	return mainContent
}

// Остальные функции без изменений
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
		status, _ := p.Status()
		user, _ := p.Username()
		cmd, _ := p.Cmdline()

		result = append(result, ProcessInfo{
			PID:     p.Pid,
			Name:    name,
			CPU:     cpu / 10,
			Memory:  float32(mem),
			Status:  status,
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
