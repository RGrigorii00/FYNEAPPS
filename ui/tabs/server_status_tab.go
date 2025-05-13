package tabs

import (
	"context"
	"fmt"
	"image/color"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ServerInfo struct {
	Name    string
	Host    string
	Status  string
	Ping    string
	Updated string
}

func CreateServerStatusTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Доступность серверов ПГАТУ", theme.ForegroundColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Инициализация данных серверов
	servers := []ServerInfo{
		{"Сайт ПГАТУ", "91.203.238.2", "Checking...", "N/A", ""},
		{"Корпоративный Портал ПГАТУ", "91.203.238.4", "Checking...", "N/A", ""},
		{"Мой хост", "83.166.245.249", "Checking...", "N/A", ""},
	}

	// Задаем фиксированные ширины столбцов
	columnWidths := []float32{
		250, // Имя сервера
		150, // Адрес
		120, // Статус
		100, // Ping
		150, // Обновлено
	}

	// Создаем заголовки таблицы
	headerRow := container.NewHBox()
	headers := []string{"Имя сервера", "Адрес", "Статус", "Ping (мс)", "Обновлено"}

	// Отступы для заголовков
	headerLeftPaddings := []float32{
		240/2 - 50, // Имя сервера
		180 / 2,    // Адрес
		160 / 2,    // Статус
		100 / 2,    // Ping
		100 / 2,    // Обновлено
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
	serverTable := widget.NewTable(
		func() (int, int) { return len(servers), len(columnWidths) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row >= len(servers) {
				label.SetText("")
				return
			}

			server := servers[id.Row]
			switch id.Col {
			case 0:
				label.SetText(server.Name)
				label.Alignment = fyne.TextAlignLeading
			case 1:
				label.SetText(server.Host)
				label.Alignment = fyne.TextAlignLeading
			case 2:
				label.SetText(server.Status)
				label.Alignment = fyne.TextAlignCenter
				updateStatusStyle(label, server.Status)
			case 3:
				label.SetText(server.Ping)
				label.Alignment = fyne.TextAlignCenter
				updatePingStyle(label, server.Ping)
			case 4:
				label.SetText(server.Updated)
				label.Alignment = fyne.TextAlignCenter
			}
		},
	)

	// Устанавливаем ширины столбцов
	for i, width := range columnWidths {
		serverTable.SetColumnWidth(i, width)
	}

	// Элементы управления
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по имени или адресу...")
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)
	autoRefreshCheck := widget.NewCheck("Автообновление (30 сек)", nil)
	autoRefreshCheck.SetChecked(true)
	sortSelect := widget.NewSelect([]string{"Имя", "Статус", "Ping", "Обновлено"}, nil)
	sortSelect.SetSelected("Имя")

	// Канал для остановки автообновления
	stopChan := make(chan struct{})

	// Функция фильтрации серверов
	filterServers := func(servers []ServerInfo, search string) []ServerInfo {
		if search == "" {
			return servers
		}

		var result []ServerInfo
		for _, s := range servers {
			if strings.Contains(strings.ToLower(s.Name), strings.ToLower(search)) ||
				strings.Contains(strings.ToLower(s.Host), strings.ToLower(search)) {
				result = append(result, s)
			}
		}
		return result
	}

	// Функция сортировки серверов
	sortServers := func(servers []ServerInfo, sortBy string) {
		switch sortBy {
		case "Имя":
			sort.Slice(servers, func(i, j int) bool {
				return servers[i].Name < servers[j].Name
			})
		case "Статус":
			sort.Slice(servers, func(i, j int) bool {
				return servers[i].Status < servers[j].Status
			})
		case "Ping":
			sort.Slice(servers, func(i, j int) bool {
				// Сначала Online, потом Offline
				if servers[i].Status != servers[j].Status {
					return servers[i].Status == "Online"
				}
				// Для Online серверов сортируем по ping
				if servers[i].Status == "Online" && servers[j].Status == "Online" {
					pingI, _ := strconv.Atoi(strings.TrimSuffix(servers[i].Ping, " ms"))
					pingJ, _ := strconv.Atoi(strings.TrimSuffix(servers[j].Ping, " ms"))
					return pingI < pingJ
				}
				return false
			})
		case "Обновлено":
			sort.Slice(servers, func(i, j int) bool {
				return servers[i].Updated > servers[j].Updated
			})
		}
	}

	// Функция обновления списка серверов
	updateServers := func() {
		for i := range servers {
			go func(index int) {
				host := servers[index].Host
				online, pingTime := pingHost(host)
				now := time.Now().Format("15:04:05")

				fyne.Do(func() {
					if online {
						servers[index].Status = "Online"
						servers[index].Ping = fmt.Sprintf("%d ms", pingTime)
					} else {
						servers[index].Status = "Offline"
						servers[index].Ping = "Timeout"
					}
					servers[index].Updated = now

					// Применяем фильтрацию и сортировку
					filtered := filterServers(servers, searchEntry.Text)
					sortServers(filtered, sortSelect.Selected)

					// Обновляем таблицу
					serverTable.Refresh()
				})
			}(i)
		}
	}

	// Автообновление
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if autoRefreshCheck.Checked {
					updateServers()
				}
			case <-stopChan:
				return
			}
		}
	}()

	// Обработчики событий
	searchEntry.OnChanged = func(s string) { updateServers() }
	refreshBtn.OnTapped = func() { updateServers() }
	autoRefreshCheck.OnChanged = func(checked bool) {
		if checked {
			updateServers()
		}
	}
	sortSelect.OnChanged = func(s string) { updateServers() }

	// Первоначальное обновление
	updateServers()

	// Очистка при закрытии
	window.SetOnClosed(func() {
		close(stopChan)
	})

	// Компоновка элементов управления
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
					autoRefreshCheck,
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
			container.NewScroll(serverTable),
		),
	)

	return mainContent
}

func pingHost(host string) (bool, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	var pingTime int

	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "1000", host)
	case "linux", "darwin":
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", host)
	default:
		return false, 0
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, 0
	}

	outputStr := strings.ToLower(string(output))
	switch runtime.GOOS {
	case "windows":
		if strings.Contains(outputStr, "ttl=") {
			timeIndex := strings.Index(outputStr, "time=")
			if timeIndex != -1 {
				timeStr := outputStr[timeIndex+5:]
				endIndex := strings.Index(timeStr, "ms")
				if endIndex != -1 {
					timeStr = timeStr[:endIndex]
					if t, err := strconv.Atoi(timeStr); err == nil {
						pingTime = t
					}
				}
			}
			return true, pingTime
		}
	case "linux", "darwin":
		if strings.Contains(outputStr, "1 packets received") || strings.Contains(outputStr, "1 received") {
			timeIndex := strings.Index(outputStr, "time=")
			if timeIndex != -1 {
				timeStr := outputStr[timeIndex+5:]
				endIndex := strings.Index(timeStr, " ms")
				if endIndex != -1 {
					timeStr = timeStr[:endIndex]
					if t, err := strconv.ParseFloat(timeStr, 64); err == nil {
						pingTime = int(t)
					}
				}
			}
			return true, pingTime
		}
	}

	return false, 0
}

func updateStatusStyle(label *widget.Label, status string) {
	switch status {
	case "Online":
		label.Importance = widget.SuccessImportance
		label.TextStyle = fyne.TextStyle{Bold: true}
	case "Offline":
		label.Importance = widget.DangerImportance
		label.TextStyle = fyne.TextStyle{Bold: true}
	default: // "Checking..."
		label.Importance = widget.WarningImportance
		label.TextStyle = fyne.TextStyle{Bold: false}
	}
	label.Refresh()
}

func updatePingStyle(label *widget.Label, ping string) {
	switch {
	case ping == "N/A" || ping == "Timeout":
		label.Importance = widget.DangerImportance
	case ping == "...":
		label.Importance = widget.WarningImportance
	default:
		label.Importance = widget.MediumImportance
	}

	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Refresh()
}
