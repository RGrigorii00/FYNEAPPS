package tabs

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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

type ServerStatusTab struct {
	servers          []ServerInfo
	filteredServers  []ServerInfo
	serverTable      *widget.Table
	searchEntry      *widget.Entry
	autoRefreshCheck *widget.Check
	sortSelect       *widget.Select
	stopChan         chan struct{}
	refreshMutex     sync.Mutex
}

func CreateServerStatusTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Доступность серверов ПГАТУ", theme.ForegroundColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	initialServers := []ServerInfo{
		{"Сайт ПГАТУ", "91.203.238.2", "Checking...", "N/A", ""},
		{"Корпоративный Портал ПГАТУ", "91.203.238.4", "Checking...", "N/A", ""},
		{"Мой хост", "83.166.245.249", "Checking...", "N/A", ""},
	}

	tab := &ServerStatusTab{
		servers:         initialServers,
		filteredServers: make([]ServerInfo, len(initialServers)),
		stopChan:        make(chan struct{}),
	}
	copy(tab.filteredServers, initialServers)

	columnWidths := []float32{250, 150, 120, 100, 150}

	// Создаем заголовки таблицы
	headerRow := container.NewHBox()
	headers := []string{"Имя сервера", "Адрес", "Статус", "Ping (мс)", "Обновлено"}

	for _, header := range headers {
		label := widget.NewLabel(header)
		label.TextStyle = fyne.TextStyle{Bold: true}
		label.Alignment = fyne.TextAlignLeading
		headerRow.Add(container.NewHBox(widget.NewLabel("  "), label))
	}

	// Создаем таблицу
	tab.serverTable = widget.NewTable(
		func() (int, int) { return len(tab.filteredServers), len(columnWidths) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignLeading
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row >= len(tab.filteredServers) {
				label.SetText("")
				return
			}

			server := tab.filteredServers[id.Row]
			switch id.Col {
			case 0:
				label.SetText("  " + server.Name)
				resetLabelStyle(label)
			case 1:
				label.SetText("  " + server.Host)
				resetLabelStyle(label)
			case 2:
				label.SetText("  " + server.Status)
				updateStatusStyle(label, server.Status) // Только здесь применяем стиль
			case 3:
				label.SetText("  " + server.Ping)
				resetLabelStyle(label)
			case 4:
				label.SetText("  " + server.Updated)
				resetLabelStyle(label)
			}
		},
	)

	for i, width := range columnWidths {
		tab.serverTable.SetColumnWidth(i, width)
	}

	// Элементы управления
	tab.searchEntry = widget.NewEntry()
	tab.searchEntry.SetPlaceHolder("Поиск по имени или адресу...")
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)
	tab.autoRefreshCheck = widget.NewCheck("Автообновление (30 сек)", nil)
	tab.autoRefreshCheck.SetChecked(true)
	tab.sortSelect = widget.NewSelect([]string{"Имя", "Статус", "Ping", "Обновлено"}, nil)
	tab.sortSelect.SetSelected("Имя")

	// Функция фильтрации серверов
	filterServers := func(search string) {
		if search == "" {
			tab.filteredServers = make([]ServerInfo, len(tab.servers))
			copy(tab.filteredServers, tab.servers)
			return
		}

		tab.filteredServers = []ServerInfo{}
		for _, s := range tab.servers {
			if strings.Contains(strings.ToLower(s.Name), strings.ToLower(search)) ||
				strings.Contains(strings.ToLower(s.Host), strings.ToLower(search)) {
				tab.filteredServers = append(tab.filteredServers, s)
			}
		}
	}

	// Функция сортировки серверов
	sortServers := func(sortBy string) {
		switch sortBy {
		case "Имя":
			sort.Slice(tab.filteredServers, func(i, j int) bool {
				return tab.filteredServers[i].Name < tab.filteredServers[j].Name
			})
		case "Статус":
			sort.Slice(tab.filteredServers, func(i, j int) bool {
				return tab.filteredServers[i].Status < tab.filteredServers[j].Status
			})
		case "Ping":
			sort.Slice(tab.filteredServers, func(i, j int) bool {
				if tab.filteredServers[i].Status != tab.filteredServers[j].Status {
					return tab.filteredServers[i].Status == "Online"
				}
				if tab.filteredServers[i].Status == "Online" && tab.filteredServers[j].Status == "Online" {
					pingI, _ := strconv.Atoi(strings.TrimSuffix(tab.filteredServers[i].Ping, " ms"))
					pingJ, _ := strconv.Atoi(strings.TrimSuffix(tab.filteredServers[j].Ping, " ms"))
					return pingI < pingJ
				}
				return false
			})
		case "Обновлено":
			sort.Slice(tab.filteredServers, func(i, j int) bool {
				return tab.filteredServers[i].Updated > tab.filteredServers[j].Updated
			})
		}
	}

	// Функция обновления списка серверов
	updateServers := func() {
		tab.refreshMutex.Lock()
		defer tab.refreshMutex.Unlock()

		var wg sync.WaitGroup
		for i := range tab.servers {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				host := tab.servers[index].Host
				online, pingTime := pingHost(host)
				now := time.Now().Format("15:04:05")

				fyne.Do(func() {
					if online {
						tab.servers[index].Status = "Online"
						tab.servers[index].Ping = fmt.Sprintf("%d ms", pingTime)
					} else {
						tab.servers[index].Status = "Offline"
						tab.servers[index].Ping = "Timeout"
					}
					tab.servers[index].Updated = now

					filterServers(tab.searchEntry.Text)
					sortServers(tab.sortSelect.Selected)
					tab.serverTable.Refresh()
				})
			}(i)
		}
		wg.Wait()
	}

	// Автообновление
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if tab.autoRefreshCheck.Checked {
					updateServers()
				}
			case <-tab.stopChan:
				return
			}
		}
	}()

	// Обработчики событий
	tab.searchEntry.OnChanged = func(s string) {
		filterServers(s)
		sortServers(tab.sortSelect.Selected)
		tab.serverTable.Refresh()
	}
	refreshBtn.OnTapped = func() { updateServers() }
	tab.autoRefreshCheck.OnChanged = func(checked bool) {
		if checked {
			updateServers()
		}
	}
	tab.sortSelect.OnChanged = func(string) {
		sortServers(tab.sortSelect.Selected)
		tab.serverTable.Refresh()
	}

	updateServers()

	window.SetOnClosed(func() {
		close(tab.stopChan)
	})

	// Компоновка интерфейса
	return container.NewBorder(
		container.NewVBox(
			title,
			container.NewBorder(
				nil, nil,
				widget.NewLabel("Поиск:"),
				container.NewHBox(
					widget.NewLabel("Сортировка:"),
					tab.sortSelect,
					tab.autoRefreshCheck,
					refreshBtn,
				),
				tab.searchEntry,
			),
		),
		nil,
		nil,
		nil,
		container.NewBorder(
			headerRow,
			nil,
			nil,
			nil,
			container.NewScroll(tab.serverTable),
		),
	)
}

// Ваши оригинальные функции:

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

// Новая вспомогательная функция для сброса стилей
func resetLabelStyle(label *widget.Label) {
	label.Importance = widget.MediumImportance
	label.TextStyle = fyne.TextStyle{}
	label.Refresh()
}
