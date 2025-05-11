package tabs

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ServerStatus представляет статус сервера
type ServerStatus struct {
	Name   string
	Host   string
	Status string
	Ping   string
}

func CreateServerStatusTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Доступность серверов ПГАТУ", theme.Color(theme.ColorNameForeground))
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Инициализация данных серверов
	servers := []ServerStatus{
		{"Сайт ПГАТУ", "91.203.238.2", "Checking...", "N/A"},
		{"Корпоративный Портал ПГАТУ", "91.203.238.4", "Checking...", "N/A"},
		{"Мой хост", "83.166.245.249", "Checking...", "N/A"},
	}

	// Создаем привязки данных
	data := make([]binding.Struct, len(servers))
	for i := range data {
		data[i] = binding.BindStruct(&servers[i])
	}

	// Создаем таблицу с улучшенным стилем
	table := widget.NewTable(
		func() (int, int) {
			return len(data), 4
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextTruncate
			item := data[id.Row]

			switch id.Col {
			case 0:
				name, _ := item.GetItem("Name")
				label.Bind(name.(binding.String))
				label.Alignment = fyne.TextAlignLeading
			case 1:
				host, _ := item.GetItem("Host")
				label.Bind(host.(binding.String))
				label.Alignment = fyne.TextAlignLeading
			case 2:
				status, _ := item.GetItem("Status")
				label.Bind(status.(binding.String))
				label.Alignment = fyne.TextAlignCenter
				updateStatusStyle(label, status.(binding.String))
			case 3:
				ping, _ := item.GetItem("Ping")
				label.Bind(ping.(binding.String))
				label.Alignment = fyne.TextAlignCenter
				updatePingStyle(label, ping.(binding.String))
			}
		},
	)

	// Настройка размеров колонок - увеличиваем первую колонку
	table.SetColumnWidth(0, 250) // Увеличили ширину первой колонки
	table.SetColumnWidth(1, 150) // Host
	table.SetColumnWidth(2, 100) // Status
	table.SetColumnWidth(3, 80)  // Ping

	// Кнопка обновления внизу
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		go checkAllServers(data)
	})
	refreshBtn.Importance = widget.MediumImportance

	// Заголовки таблицы
	headers := container.NewGridWithColumns(4,
		widget.NewLabelWithStyle("Имя сервера", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Адрес", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Статус", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Ping (мс)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)

	// Первоначальная проверка серверов
	go checkAllServers(data)

	// Автоматическое обновление каждые 30 секунд
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			checkAllServers(data)
		}
	}()

	// Создаем контейнер с прокруткой для таблицы
	scrollContainer := container.NewScroll(table)
	scrollContainer.SetMinSize(fyne.NewSize(650, 400))

	// Основной контейнер
	mainContent := container.NewBorder(
		container.NewVBox(
			title,
			widget.NewSeparator(),
			headers,
		),
		container.NewCenter(refreshBtn),
		nil,
		nil,
		scrollContainer,
	)

	return mainContent
}

func checkAllServers(data []binding.Struct) {
	for i := range data {
		go func(index int) {
			item := data[index]
			host, _ := item.GetItem("Host")
			hostStr, _ := host.(binding.String).Get()

			// Устанавливаем статус "Checking..."
			status, _ := item.GetItem("Status")
			status.(binding.String).Set("Checking...")

			ping, _ := item.GetItem("Ping")
			ping.(binding.String).Set("...")

			// Выполняем ping и получаем время
			online, pingTime := pingHost(hostStr)

			// Обновляем статус и ping
			if online {
				status.(binding.String).Set("Online")
				ping.(binding.String).Set(fmt.Sprintf("%d ms", pingTime))
			} else {
				status.(binding.String).Set("Offline")
				ping.(binding.String).Set("Timeout")
			}
		}(i)
	}
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

func updateStatusStyle(label *widget.Label, status binding.String) {
	statusStr, _ := status.Get()
	switch statusStr {
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

func updatePingStyle(label *widget.Label, ping binding.String) {
	pingStr, _ := ping.Get()

	switch {
	case pingStr == "N/A" || pingStr == "Timeout":
		label.Importance = widget.DangerImportance
	case pingStr == "...":
		label.Importance = widget.WarningImportance
	default:
		// Стиль для нормальных значений ping
		label.Importance = widget.MediumImportance
	}

	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Refresh()
}
