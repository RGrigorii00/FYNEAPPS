package tabs

import (
	"context"
	"math/rand"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// ServerStatus представляет статус сервера
type ServerStatus struct {
	Name   string
	Host   string // Изменено с URL на Host для ping
	Status string
	Load   int
}

func CreateServerStatusTab(window fyne.Window) fyne.CanvasObject {
	title := widget.NewLabel("Server Monitoring")
	title.TextStyle = fyne.TextStyle{Bold: true, Italic: true}
	title.Alignment = fyne.TextAlignCenter

	// Инициализация данных серверов
	servers := []ServerStatus{
		{"Main API", "91.203.238.2", "Checking...", 0},
		{"Database", "91.203.238.4", "Checking...", 0},
		{"Cache", "83.166.245.249", "Checking...", 0},
	}

	// Создаем привязки данных
	data := make([]binding.Struct, len(servers))
	for i := range data {
		data[i] = binding.BindStruct(&servers[i])
	}

	// Создаем таблицу
	table := widget.NewTable(
		func() (int, int) {
			return len(data), 4
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			item := data[id.Row]

			switch id.Col {
			case 0:
				name, _ := item.GetItem("Name")
				label.Bind(name.(binding.String))
			case 1:
				host, _ := item.GetItem("Host")
				label.Bind(host.(binding.String))
			case 2:
				status, _ := item.GetItem("Status")
				label.Bind(status.(binding.String))
				updateStatusStyle(label, status.(binding.String))
			case 3:
				load, _ := item.GetItem("Load")
				label.Bind(binding.IntToString(load.(binding.Int)))
				updateLoadStyle(label, load.(binding.Int))
			}
		},
	)

	// Настройка размеров колонок
	table.SetColumnWidth(0, 200)
	table.SetColumnWidth(1, 250)
	table.SetColumnWidth(2, 150)
	table.SetColumnWidth(3, 100)

	// Кнопка обновления
	refreshBtn := widget.NewButton("Refresh Status", func() {
		checkAllServers(data)
	})
	refreshBtn.Importance = widget.HighImportance

	// Заголовки таблицы
	headers := container.NewGridWithColumns(4,
		widget.NewLabelWithStyle("Server Name", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Host", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Status", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Load %", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
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
	scrollContainer.SetMinSize(fyne.NewSize(800, 500))

	// Основной контейнер
	mainContent := container.NewBorder(
		container.NewVBox(
			title,
			container.NewCenter(refreshBtn),
			headers,
		),
		nil,
		nil,
		nil,
		scrollContainer,
	)

	return mainContent
}

// checkAllServers проверяет статус всех серверов
func checkAllServers(data []binding.Struct) {
	for i := range data {
		go func(index int) {
			item := data[index]
			host, _ := item.GetItem("Host")
			hostStr, _ := host.(binding.String).Get()

			// Устанавливаем статус "Checking..."
			status, _ := item.GetItem("Status")
			status.(binding.String).Set("Checking...")

			// Выполняем ping
			online := pingHost(hostStr)

			// Обновляем статус
			if online {
				status.(binding.String).Set("Online")
				// Для ping нагрузку можно имитировать
				load, _ := item.GetItem("Load")
				load.(binding.Int).Set(rand.Intn(100))
			} else {
				status.(binding.String).Set("Offline")
				load, _ := item.GetItem("Load")
				load.(binding.Int).Set(0)
			}
		}(i)
	}
}

// pingHost выполняет ping указанного хоста
func pingHost(host string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	// Определяем команду ping в зависимости от ОС
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "1000", host)
	case "linux", "darwin":
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", host)
	default:
		return false
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Анализируем вывод команды ping
	outputStr := strings.ToLower(string(output))
	switch runtime.GOOS {
	case "windows":
		return strings.Contains(outputStr, "ttl=")
	case "linux", "darwin":
		return strings.Contains(outputStr, "1 packets received") ||
			strings.Contains(outputStr, "1 received")
	default:
		return false
	}
}

// Остальные функции остаются без изменений
func updateStatusStyle(label *widget.Label, status binding.String) {
	statusStr, _ := status.Get()
	if statusStr == "Online" {
		label.Importance = widget.SuccessImportance
		label.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		label.Importance = widget.DangerImportance
		label.TextStyle = fyne.TextStyle{Bold: false}
	}
	label.Refresh()
}

func updateLoadStyle(label *widget.Label, load binding.Int) {
	loadVal, _ := load.Get()
	if loadVal > 70 {
		label.Importance = widget.WarningImportance
	} else if loadVal > 30 {
		label.Importance = widget.MediumImportance
	} else {
		label.Importance = widget.LowImportance
	}
	label.Refresh()
}

// updateServerData обновляет данные серверов
func updateServerData(data []binding.Struct) {
	for i := range data {
		// Обновляем статус случайным образом
		newStatus := "Online"
		if rand.Intn(10) < 2 { // 20% chance for offline
			newStatus = "Offline"
		}
		status, _ := data[i].GetItem("Status")
		status.(binding.String).Set(newStatus)

		// Обновляем нагрузку
		if newStatus == "Online" {
			load, _ := data[i].GetItem("Load")
			load.(binding.Int).Set(rand.Intn(100))
		} else {
			load, _ := data[i].GetItem("Load")
			load.(binding.Int).Set(0)
		}
	}
}

// autoRefresh автоматически обновляет данные через заданный интервал
func autoRefresh(data []binding.Struct, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		updateServerData(data)
	}
}
