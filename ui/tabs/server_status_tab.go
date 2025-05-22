package tabs

import (
	"context"
	"fmt"
	"net/http"
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
	HTTP    string
	Trace   string
	Updated string
}

type ServerStatusTab struct {
	servers          []ServerInfo
	filteredServers  []ServerInfo
	serverTable      *widget.Table
	searchEntry      *widget.Entry
	autoRefreshCheck *widget.Check
	sortSelect       *widget.Select
	ctx              context.Context
	cancel           context.CancelFunc
	refreshMutex     sync.Mutex
	wg               sync.WaitGroup
}

func CreateServerStatusTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Доступность серверов ПГАТУ", theme.ForegroundColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	initialServers := []ServerInfo{
		{"Сайт ПГАТУ", "91.203.238.2", "Checking...", "N/A", "N/A", "N/A", ""},
		{"Корпоративный Портал ПГАТУ", "91.203.238.4", "Checking...", "N/A", "N/A", "N/A", ""},
		{"Мой хост", "83.166.245.249", "Checking...", "N/A", "N/A", "N/A", ""},
	}

	ctx, cancel := context.WithCancel(context.Background())

	tab := &ServerStatusTab{
		servers:         initialServers,
		filteredServers: make([]ServerInfo, len(initialServers)),
		ctx:             ctx,
		cancel:          cancel,
	}
	copy(tab.filteredServers, initialServers)

	columnWidths := []float32{200, 120, 100, 80, 80, 150, 120}

	headerRow := container.NewHBox()
	headers := []string{"Имя сервера", "Адрес", "Статус", "Ping", "HTTP", "Trace", "Обновлено"}

	for _, header := range headers {
		label := widget.NewLabel(header)
		label.TextStyle = fyne.TextStyle{Bold: true}
		label.Alignment = fyne.TextAlignLeading
		headerRow.Add(container.NewHBox(widget.NewLabel("  "), label))
	}

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
				updateStatusStyle(label, server.Status)
			case 3:
				label.SetText("  " + server.Ping)
				resetLabelStyle(label)
			case 4:
				label.SetText("  " + server.HTTP)
				updateHTTPStatusStyle(label, server.HTTP)
			case 5:
				label.SetText("  " + server.Trace)
				updateTraceStyle(label, server.Trace)
			case 6:
				label.SetText("  " + server.Updated)
				resetLabelStyle(label)
			}
		},
	)

	for i, width := range columnWidths {
		tab.serverTable.SetColumnWidth(i, width)
	}

	tab.searchEntry = widget.NewEntry()
	tab.searchEntry.SetPlaceHolder("Поиск по имени или адресу...")

	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)
	tab.autoRefreshCheck = widget.NewCheck("Автообновление (30 сек)", nil)
	tab.autoRefreshCheck.SetChecked(false)
	tab.sortSelect = widget.NewSelect([]string{"Имя", "Статус", "Ping", "HTTP", "Trace", "Обновлено"}, nil)
	tab.sortSelect.SetSelected("Имя")

	filterServers := func(search string) {
		fyne.Do(func() {
			if search == "" {
				tab.filteredServers = make([]ServerInfo, len(tab.servers))
				copy(tab.filteredServers, tab.servers)
				return
			}

			var filtered []ServerInfo
			for _, s := range tab.servers {
				if strings.Contains(strings.ToLower(s.Name), strings.ToLower(search)) ||
					strings.Contains(strings.ToLower(s.Host), strings.ToLower(search)) {
					filtered = append(filtered, s)
				}
			}
			tab.filteredServers = filtered
		})
	}

	sortServers := func(sortBy string) {
		fyne.Do(func() {
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
					return extractNumericValue(tab.filteredServers[i].Ping) < extractNumericValue(tab.filteredServers[j].Ping)
				})
			case "HTTP":
				sort.Slice(tab.filteredServers, func(i, j int) bool {
					return extractNumericValue(tab.filteredServers[i].HTTP) < extractNumericValue(tab.filteredServers[j].HTTP)
				})
			case "Trace":
				sort.Slice(tab.filteredServers, func(i, j int) bool {
					return extractNumericValue(tab.filteredServers[i].Trace) < extractNumericValue(tab.filteredServers[j].Trace)
				})
			case "Обновлено":
				sort.Slice(tab.filteredServers, func(i, j int) bool {
					return tab.filteredServers[i].Updated > tab.filteredServers[j].Updated
				})
			}
		})
	}

	updateServers := func() {
		tab.refreshMutex.Lock()
		defer tab.refreshMutex.Unlock()

		select {
		case <-tab.ctx.Done():
			return
		default:
		}

		fyne.Do(func() {
			for i := range tab.servers {
				tab.servers[i].Status = "Checking..."
				tab.servers[i].Ping = "N/A"
				tab.servers[i].HTTP = "N/A"
				tab.servers[i].Trace = "N/A"
				tab.servers[i].Updated = time.Now().Format("15:04:05")
			}
			filterServers(tab.searchEntry.Text)
			tab.serverTable.Refresh()
		})

		var wg sync.WaitGroup
		for i := range tab.servers {
			wg.Add(1)
			tab.wg.Add(1)

			go func(index int) {
				defer wg.Done()
				defer tab.wg.Done()

				select {
				case <-tab.ctx.Done():
					return
				default:
					host := tab.servers[index].Host
					now := time.Now().Format("15:04:05")

					pingOnline, pingTime := pingHost(tab.ctx, host)
					httpStatus := checkHTTP(tab.ctx, host)
					traceResult := traceHost(tab.ctx, host)

					fyne.Do(func() {
						select {
						case <-tab.ctx.Done():
							return
						default:
							if pingOnline {
								tab.servers[index].Status = "Online"
								tab.servers[index].Ping = fmt.Sprintf("%d ms", pingTime)
							} else {
								tab.servers[index].Status = "Offline"
								tab.servers[index].Ping = "Timeout"
							}
							tab.servers[index].HTTP = httpStatus
							tab.servers[index].Trace = traceResult
							tab.servers[index].Updated = now

							filterServers(tab.searchEntry.Text)
							sortServers(tab.sortSelect.Selected)
							tab.serverTable.Refresh()
						}
					})
				}
			}(i)
		}
		wg.Wait()
	}

	refreshBtn.OnTapped = func() { go updateServers() }
	tab.searchEntry.OnChanged = func(s string) {
		filterServers(s)
		sortServers(tab.sortSelect.Selected)
		fyne.Do(func() {
			tab.serverTable.Refresh()
		})
	}
	tab.autoRefreshCheck.OnChanged = func(checked bool) {
		if checked {
			go updateServers()
		}
	}
	tab.sortSelect.OnChanged = func(string) {
		sortServers(tab.sortSelect.Selected)
		fyne.Do(func() {
			tab.serverTable.Refresh()
		})
	}

	tab.wg.Add(1)
	go func() {
		defer tab.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if tab.autoRefreshCheck.Checked {
					go updateServers()
				}
			case <-tab.ctx.Done():
				return
			}
		}
	}()

	tab.wg.Add(1)
	go func() {
		defer tab.wg.Done()
		select {
		case <-time.After(500 * time.Millisecond):
			updateServers()
		case <-tab.ctx.Done():
			return
		}
	}()

	window.SetOnClosed(func() {
		tab.Close()
	})

	return container.NewBorder(
		container.NewVBox(
			title,
			widget.NewSeparator(),
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
			container.NewVBox(
				widget.NewSeparator(),
				headerRow,
			),
			nil,
			nil,
			nil,
			container.NewScroll(tab.serverTable),
		),
	)
}

func (tab *ServerStatusTab) Close() {
	tab.cancel()
	tab.wg.Wait()
	tab.refreshMutex.Lock()
	defer tab.refreshMutex.Unlock()
}

func pingHost(ctx context.Context, host string) (bool, int) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
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

func checkHTTP(ctx context.Context, host string) string {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	req, err := http.NewRequestWithContext(ctx, "GET", host, nil)
	if err != nil {
		return "No HTTP"
	}

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "No HTTP"
	}
	defer resp.Body.Close()

	return fmt.Sprintf("HTTP %d", resp.StatusCode)
}

func traceHost(ctx context.Context, host string) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "tracert", "-d", "-h", "5", host)
	case "linux", "darwin":
		cmd = exec.CommandContext(ctx, "traceroute", "-m", "5", host)
	default:
		return "N/A"
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Trace err"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 1 {
		return fmt.Sprintf("%d hops", len(lines)-2)
	}

	return "Trace err"
}

func extractNumericValue(s string) int {
	if s == "N/A" || s == "Timeout" || s == "No HTTP" || s == "Trace err" {
		return 9999
	}

	parts := strings.Fields(s)
	if len(parts) > 0 {
		if num, err := strconv.Atoi(parts[0]); err == nil {
			return num
		}
	}

	return 9999
}

func updateStatusStyle(label *widget.Label, status string) {
	fyne.Do(func() {
		switch status {
		case "Online":
			label.Importance = widget.SuccessImportance
			label.TextStyle = fyne.TextStyle{Bold: true}
		case "Offline":
			label.Importance = widget.DangerImportance
			label.TextStyle = fyne.TextStyle{Bold: true}
		default:
			label.Importance = widget.WarningImportance
			label.TextStyle = fyne.TextStyle{Bold: false}
		}
		label.Refresh()
	})
}

func updateHTTPStatusStyle(label *widget.Label, status string) {
	fyne.Do(func() {
		if strings.HasPrefix(status, "HTTP 2") || strings.HasPrefix(status, "HTTP 3") {
			label.Importance = widget.SuccessImportance
		} else if strings.HasPrefix(status, "HTTP 4") || strings.HasPrefix(status, "HTTP 5") {
			label.Importance = widget.DangerImportance
		} else if status == "No HTTP" {
			label.Importance = widget.WarningImportance
		} else {
			label.Importance = widget.MediumImportance
		}
		label.Refresh()
	})
}

func updateTraceStyle(label *widget.Label, trace string) {
	fyne.Do(func() {
		if strings.Contains(trace, "hops") {
			label.Importance = widget.SuccessImportance
		} else if trace == "Trace err" {
			label.Importance = widget.DangerImportance
		} else {
			label.Importance = widget.MediumImportance
		}
		label.Refresh()
	})
}

func resetLabelStyle(label *widget.Label) {
	fyne.Do(func() {
		label.Importance = widget.MediumImportance
		label.TextStyle = fyne.TextStyle{}
		label.Refresh()
	})
}
