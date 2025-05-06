package tabs

import (
	"fmt"
	"image/color"
	"log"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type HardwareComponent struct {
	ID      string
	Name    string
	Icon    fyne.Resource
	Usage   float64
	Details map[string]string
	Disks   []DiskInfo
}

type DiskInfo struct {
	Name      string
	Total     uint64
	Used      uint64
	Free      uint64
	Usage     float64
	MountPath string
}

func CreateHardwareTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Мониторинг системы", theme.Color(theme.ColorNameForeground))
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Создаем контейнер для карточек
	cardsContainer := container.NewGridWithColumns(2)
	scrollContainer := container.NewVScroll(
		container.NewVBox(
			container.NewPadded(title),
			widget.NewSeparator(),
			cardsContainer,
		),
	)
	scrollContainer.SetMinSize(fyne.NewSize(800, 600))

	// Функция обновления данных
	updateData := func() {
		components, err := getSystemComponents()
		if err != nil {
			log.Printf("Error getting system info: %v", err)
			return
		}

		fyne.Do(func() {
			cardsContainer.Objects = make([]fyne.CanvasObject, 0) // Более безопасная очистка
			for _, component := range components {
				card := createHardwareCard(component)
				paddedCard := container.NewPadded(card)
				cardsContainer.Add(paddedCard)
			}
			cardsContainer.Refresh()
		})
	}

	// Первоначальное обновление
	updateData()

	// Запускаем автообновление
	autoRefresh := time.NewTicker(2 * time.Second)
	go func() {
		for range autoRefresh.C {
			updateData()
		}
	}()

	// Кнопка ручного обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), updateData)
	refreshBtn.Importance = widget.MediumImportance

	// Очистка при закрытии
	window.SetOnClosed(func() {
		autoRefresh.Stop()
	})

	return container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), refreshBtn, layout.NewSpacer()),
		nil,
		nil,
		scrollContainer,
	)
}

func getSystemComponents() ([]HardwareComponent, error) {
	var components []HardwareComponent

	// CPU информация
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting CPU info: %v", err)
	}
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("error getting CPU usage: %v", err)
	}

	cpuDetails := map[string]string{
		"Модель":        cpuInfo[0].ModelName,
		"Ядра":          fmt.Sprintf("%d (%d потоков)", runtime.NumCPU(), cpuInfo[0].Cores),
		"Частота":       fmt.Sprintf("%.2f GHz", float64(cpuInfo[0].Mhz)/1000),
		"Производитель": cpuInfo[0].VendorID,
	}

	components = append(components, HardwareComponent{
		ID:      "cpu",
		Name:    "Процессор",
		Icon:    theme.ComputerIcon(),
		Usage:   cpuUsage[0],
		Details: cpuDetails,
	})

	// Память
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error getting memory info: %v", err)
	}

	memDetails := map[string]string{
		"Всего":        fmt.Sprintf("%.2f GB", float64(memInfo.Total)/1024/1024/1024),
		"Использовано": fmt.Sprintf("%.2f GB", float64(memInfo.Used)/1024/1024/1024),
		"Свободно":     fmt.Sprintf("%.2f GB", float64(memInfo.Free)/1024/1024/1024),
	}

	components = append(components, HardwareComponent{
		ID:      "memory",
		Name:    "Оперативная память",
		Icon:    theme.StorageIcon(),
		Usage:   memInfo.UsedPercent,
		Details: memDetails,
	})

	// Диски (отдельная карточка с подразделами)
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("error getting disk info: %v", err)
	}

	var disks []DiskInfo
	for _, part := range partitions {
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			continue
		}

		disks = append(disks, DiskInfo{
			Name:      part.Device,
			Total:     usage.Total,
			Used:      usage.Used,
			Free:      usage.Free,
			Usage:     usage.UsedPercent,
			MountPath: part.Mountpoint,
		})
	}

	components = append(components, HardwareComponent{
		ID:    "disks",
		Name:  "Диски",
		Icon:  theme.StorageIcon(),
		Usage: 0, // Общий процент не рассчитывается
		Details: map[string]string{
			"Количество": fmt.Sprintf("%d", len(disks)),
		},
		Disks: disks,
	})

	// Сеть
	netInfo, err := net.IOCounters(false)
	if err != nil {
		return nil, fmt.Errorf("error getting network info: %v", err)
	}

	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting host info: %v", err)
	}

	netDetails := map[string]string{
		"Имя хоста":     hostInfo.Hostname,
		"Отправлено":    fmt.Sprintf("%.2f MB", float64(netInfo[0].BytesSent)/1024/1024),
		"Получено":      fmt.Sprintf("%.2f MB", float64(netInfo[0].BytesRecv)/1024/1024),
		"Пакеты (отпр)": fmt.Sprintf("%d", netInfo[0].PacketsSent),
		"Пакеты (пол)":  fmt.Sprintf("%d", netInfo[0].PacketsRecv),
	}

	components = append(components, HardwareComponent{
		ID:      "network",
		Name:    "Сетевая активность",
		Icon:    theme.StorageIcon(),
		Usage:   0,
		Details: netDetails,
	})

	return components, nil
}

func createHardwareCard(component HardwareComponent) fyne.CanvasObject {
	nameLabel := widget.NewLabel(component.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel.Alignment = fyne.TextAlignCenter

	// Прогресс-бар для компонентов с показателем использования
	var usageDisplay fyne.CanvasObject
	if component.Usage > 0 {
		usageLabel := widget.NewLabel(fmt.Sprintf("Использование: %.1f%%", component.Usage))
		usageBar := widget.NewProgressBar()
		usageBar.SetValue(component.Usage / 100)
		usageBar.TextFormatter = func() string {
			return fmt.Sprintf("%.1f%%", component.Usage)
		}
		usageDisplay = container.NewVBox(usageLabel, usageBar)
	} else {
		usageDisplay = layout.NewSpacer()
	}

	// Основной контейнер для деталей
	detailsContainer := container.NewVBox()

	// Добавляем обычные детали
	for key, value := range component.Details {
		detailRow := container.NewHBox(
			widget.NewLabel(fmt.Sprintf("%s:", key)),
			layout.NewSpacer(),
			widget.NewLabel(value),
		)
		detailsContainer.Add(detailRow)
	}

	// Особый случай для дисков - добавляем подразделы
	if component.ID == "disks" && len(component.Disks) > 0 {
		detailsContainer.Add(widget.NewSeparator())
		detailsContainer.Add(widget.NewLabel("Подробно о дисках:"))

		for _, disk := range component.Disks {
			diskBar := widget.NewProgressBar()
			diskBar.SetValue(disk.Usage / 100)

			diskCard := container.NewVBox(
				widget.NewLabel(fmt.Sprintf("Диск: %s (%s)", disk.Name, disk.MountPath)),
				container.NewHBox(
					widget.NewLabel("Использовано:"),
					layout.NewSpacer(),
					widget.NewLabel(fmt.Sprintf("%.1f%% (%.1f GB из %.1f GB)",
						disk.Usage,
						float64(disk.Used)/1024/1024/1024,
						float64(disk.Total)/1024/1024/1024)),
				),
				diskBar,
				widget.NewSeparator(),
			)
			detailsContainer.Add(diskCard)
		}
	}

	// Основное содержимое карточки
	cardContent := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			widget.NewIcon(component.Icon),
			layout.NewSpacer(),
		),
		container.NewCenter(nameLabel),
		widget.NewSeparator(),
		usageDisplay,
		widget.NewSeparator(),
		detailsContainer,
	)

	// Стилизация карточки
	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	cardBackground.CornerRadius = 12
	cardBackground.StrokeColor = theme.Color(theme.ColorNameSeparator)
	cardBackground.StrokeWidth = 1

	shadow := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 30})
	shadow.CornerRadius = 12

	card := container.NewStack(
		container.NewPadded(shadow),
		container.NewStack(
			cardBackground,
			container.NewPadded(cardContent),
		),
	)

	return container.NewPadded(card)
}
