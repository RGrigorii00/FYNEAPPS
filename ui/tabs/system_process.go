package tabs

import (
	"fmt"
	"image/color"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	title := canvas.NewText("Процессы компьютера", theme.ForegroundColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Задаем фиксированные ширины столбцов (можно менять эти значения)
	columnWidths := []float32{
		80,  // PID
		610, // Имя процесса
		80,  // CPU %
		100, // Память
		120, // Статус
		150, // Пользователь
	}

	// Создаем заголовки таблицы
	headerRow := container.NewHBox()
	headers := []string{"PID", "Имя процесса", "CPU %", "Память", "Статус", "Пользователь"}

	// Указываем индивидуальные отступы ОТ ЛЕВОГО КРАЯ для каждого заголовка
	headerLeftPaddings := []float32{
		20,  // PID - 10px от левого края
		270, // Имя процесса - 150px от левого края
		230, // CPU % - 300px от левого края
		20,  // Память - 400px от левого края
		40,  // Статус - 500px от левого края
		50,  // Пользователь - 600px от левого края
	}

	for i, header := range headers {

		label := widget.NewLabel(header)
		label.TextStyle = fyne.TextStyle{Bold: true}
		label.Alignment = fyne.TextAlignLeading // Заголовки по левому краю

		// Создаем контейнер с абсолютным позиционированием
		paddedHeader := container.NewHBox()

		// Добавляем отступ слева
		if i < len(headerLeftPaddings) {
			// Создаем невидимый элемент для отступа
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(headerLeftPaddings[i], 1))
			paddedHeader.Add(spacer)
		}

		// Добавляем сам заголовок
		paddedHeader.Add(label)

		headerRow.Add(paddedHeader)
	}

	// Создаем таблицу
	processTable := widget.NewTable(
		func() (int, int) { return 0, len(columnWidths) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignLeading // Выравнивание по левому краю
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			// Заполнение будет в updateProcesses
		},
	)

	// Устанавливаем ширины столбцов
	for i, width := range columnWidths {
		processTable.SetColumnWidth(i, width)
	}

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
				return len(filtered), len(columnWidths)
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
					label.SetText(proc.User)
				}
				label.Alignment = fyne.TextAlignLeading // Все ячейки по левому краю
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
	controls := container.NewBorder(
		nil, nil, nil, nil,
		container.NewVBox(
			title,
			widget.NewSeparator(),
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

	mainContent := container.NewBorder(
		// Верхняя часть - все элементы выше таблицы
		container.NewVBox(
			controls,
			widget.NewSeparator(),
		),
		nil, // Нижняя часть - пустая
		nil, // Левая часть - пустая
		nil, // Правая часть - пустая
		// Основное содержимое - таблица
		container.NewBorder(
			headerRow, // Заголовки таблицы
			nil,
			nil,
			nil,
			container.NewScroll(processTable),
		),
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
