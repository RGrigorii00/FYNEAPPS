package tabs

import (
	"fmt"
	"image/color"
	"log"
	"os/exec"
	"os/user"
	"runtime"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
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

// Глобальные переменные для хранения предыдущих значений
var (
	prevNetStats []net.IOCountersStat
	prevNetTime  time.Time
)

func CreateHardwareTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Мониторинг системы", theme.Color(theme.ColorNameForeground))
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Создаем контейнер для карточек (теперь 3 в ряд)
	cardsContainer := container.NewGridWithColumns(3) // Изменено с 2 на 3
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
			cardsContainer.Objects = make([]fyne.CanvasObject, 0)
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

	autoRefresh := time.NewTicker(1 * time.Second)
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

	// Определяем разрядность процессора
	arch := "32-bit"
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		arch = "64-bit"
	}

	cpuDetails := map[string]string{
		"Модель":        cpuInfo[0].ModelName,
		"Ядра":          fmt.Sprintf("%d (%d потоков)", runtime.NumCPU(), cpuInfo[0].Cores),
		"Частота":       fmt.Sprintf("%.2f GHz", float64(cpuInfo[0].Mhz)/1000),
		"Производитель": cpuInfo[0].VendorID,
		"Разрядность":   arch,
		"Кэш L1":        fmt.Sprintf("%d KB", cpuInfo[0].CacheSize/1024),
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

	// Получаем информацию о модулях памяти (только для Linux)
	var memModules string
	if runtime.GOOS == "linux" {
		out, err := exec.Command("dmidecode", "-t", "17").Output()
		if err == nil {
			memModules = parseMemoryModules(string(out))
		}
	}

	memDetails := map[string]string{
		"Всего":        fmt.Sprintf("%.2f GB", float64(memInfo.Total)/1024/1024/1024),
		"Использовано": fmt.Sprintf("%.2f GB", float64(memInfo.Used)/1024/1024/1024),
		"Свободно":     fmt.Sprintf("%.2f GB", float64(memInfo.Free)/1024/1024/1024),
		"Тип":          getMemoryType(),
	}

	if memModules != "" {
		memDetails["Модули"] = memModules
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
	netInterfaces, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("error getting network info: %v", err)
	}

	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting host info: %v", err)
	}

	// Получаем дополнительную информацию о сетевых интерфейсах
	netStats, err := net.Interfaces()
	if err != nil {
		log.Printf("Ошибка получения информации о сетевых интерфейсах: %v", err)
	}

	// Создаем детализированную информацию о сети
	netDetails := map[string]string{
		"Имя хоста": hostInfo.Hostname,
	}

	// Добавляем общую статистику по сети
	if len(netInterfaces) > 0 {
		var totalSent, totalRecv, totalPacketsSent, totalPacketsRecv uint64
		var activeAdapters []string

		now := time.Now()

		// Рассчитываем скорости, если есть предыдущие данные
		if prevNetStats != nil && !prevNetTime.IsZero() {
			elapsed := now.Sub(prevNetTime).Seconds()

			for _, iface := range netInterfaces {
				if iface.BytesSent+iface.BytesRecv > 0 {
					// Ищем предыдущие значения для этого интерфейса
					var prevBytesSent, prevBytesRecv uint64
					for _, prevIface := range prevNetStats {
						if prevIface.Name == iface.Name {
							prevBytesSent = prevIface.BytesSent
							prevBytesRecv = prevIface.BytesRecv
							break
						}
					}

					// Рассчитываем скорости в МБ/с
					sentSpeed := float64(iface.BytesSent-prevBytesSent) / (1024 * 1024) / elapsed
					recvSpeed := float64(iface.BytesRecv-prevBytesRecv) / (1024 * 1024) / elapsed

					// Находим MAC-адрес
					var macAddr string
					for _, stat := range netStats {
						if stat.Name == iface.Name {
							macAddr = stat.HardwareAddr
							break
						}
					}

					if macAddr == "" {
						macAddr = "Недоступно"
					}

					activeAdapters = append(activeAdapters,
						fmt.Sprintf("%s (MAC: %s)\n  ▲ %.2f MB/s  ▼ %.2f MB/s",
							iface.Name, macAddr, sentSpeed, recvSpeed))
				}
			}
		} else {
			// Первый запуск - просто собираем информацию об интерфейсах
			for _, iface := range netInterfaces {
				if iface.BytesSent+iface.BytesRecv > 0 {
					var macAddr string
					for _, stat := range netStats {
						if stat.Name == iface.Name {
							macAddr = stat.HardwareAddr
							break
						}
					}

					if macAddr == "" {
						macAddr = "Недоступно"
					}

					activeAdapters = append(activeAdapters,
						fmt.Sprintf("%s (MAC: %s)\n  ▲ вычисляется...  ▼ вычисляется...",
							iface.Name, macAddr))
				}
			}
		}

		// Обновляем общую статистику
		for _, iface := range netInterfaces {
			totalSent += iface.BytesSent
			totalRecv += iface.BytesRecv
			totalPacketsSent += iface.PacketsSent
			totalPacketsRecv += iface.PacketsRecv
		}

		netDetails["Отправлено"] = fmt.Sprintf("%.2f MB", float64(totalSent)/1024/1024)
		netDetails["Получено"] = fmt.Sprintf("%.2f MB", float64(totalRecv)/1024/1024)
		netDetails["Пакеты (отпр)"] = fmt.Sprintf("%d", totalPacketsSent)
		netDetails["Пакеты (пол)"] = fmt.Sprintf("%d", totalPacketsRecv)

		// Добавляем список активных адаптеров с MAC и скоростями
		if len(activeAdapters) > 0 {
			netDetails["Активные адаптеры"] = strings.Join(activeAdapters, "\n\n")
		}

		// Сохраняем текущие значения для следующего расчета
		prevNetStats = netInterfaces
		prevNetTime = now
	}

	components = append(components, HardwareComponent{
		ID:      "network",
		Name:    "Сетевая активность",
		Icon:    theme.StorageIcon(),
		Usage:   0,
		Details: netDetails,
	})

	// Системная информация
	sysDetails := map[string]string{
		"ОС":                   hostInfo.OS,
		"Платформа":            hostInfo.Platform,
		"Версия ОС":            hostInfo.PlatformVersion,
		"Архитектура":          hostInfo.KernelArch,
		"Время работы":         fmt.Sprintf("%v", time.Duration(hostInfo.Uptime)*time.Second),
		"Количество процессов": fmt.Sprintf("%d", hostInfo.Procs),
	}

	components = append(components, HardwareComponent{
		ID:      "system",
		Name:    "Система",
		Icon:    theme.SettingsIcon(),
		Usage:   0,
		Details: sysDetails,
	})

	currentUser, err := user.Current()
	userDetails := map[string]string{
		"Имя хоста":      hostInfo.Hostname,
		"Время загрузки": fmt.Sprintf("%v", time.Unix(int64(hostInfo.BootTime), 0).Format("2006-01-02 15:04:05")),
		"Версия ядра":    hostInfo.KernelVersion,
		"ID хоста":       hostInfo.HostID,
	}

	// Добавляем информацию о пользователе, если удалось ее получить
	if err == nil {
		// Имя пользователя
		username := currentUser.Username
		if runtime.GOOS == "windows" {
			if backslashPos := strings.Index(username, "\\"); backslashPos != -1 {
				username = username[backslashPos+1:]
			}
		}
		userDetails["Пользователь"] = username

		// Имя группы
		group, err := user.LookupGroupId(currentUser.Gid)
		if err == nil {
			userDetails["Группа"] = group.Name
		} else {
			userDetails["Группа"] = "Недоступно"
		}

		userDetails["UID"] = currentUser.Uid
		userDetails["GID"] = currentUser.Gid
		userDetails["Домашняя папка"] = currentUser.HomeDir
	} else {
		log.Printf("Ошибка получения информации о пользователе: %v", err)
		userDetails["Пользователь"] = "Недоступно"
		userDetails["Группа"] = "Недоступно"
		userDetails["UID"] = "Недоступно"
		userDetails["GID"] = "Недоступно"
	}

	// // Добавляем IP-адреса
	// ifaces, err := net.Interfaces()
	// if err == nil {
	// 	var ips []string
	// 	for _, i := range ifaces {
	// 		addrs, err := i.Addr()
	// 		if err != nil {
	// 			continue
	// 		}
	// 		for _, addr := range addrs {
	// 			var ip net.IP
	// 			switch v := addr.(type) {
	// 			case *net.IPNet:
	// 				ip = v.IP
	// 			case *net.IPAddr:
	// 				ip = v.IP
	// 			}
	// 			if ip.IsLoopback() {
	// 				continue
	// 			}
	// 			ips = append(ips, ip.String())
	// 		}
	// 	}
	// 	if len(ips) > 0 {
	// 		userDetails["IP-адреса"] = strings.Join(ips, ", ")
	// 	}
	// }

	components = append(components, HardwareComponent{
		ID:      "user",
		Name:    "Пользователь",
		Icon:    theme.AccountIcon(),
		Usage:   0,
		Details: userDetails,
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

	// Сортируем ключи деталей для стабильного порядка
	var keys []string
	for key := range component.Details {
		keys = append(keys, key)
	}
	sort.Strings(keys) // Сортируем ключи для стабильного порядка

	// Добавляем обычные детали в отсортированном порядке
	for _, key := range keys {
		value := component.Details[key]
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

// Вспомогательная функция для определения типа памяти
func getMemoryType() string {
	if runtime.GOOS == "linux" {
		out, err := exec.Command("dmidecode", "-t", "17").Output()
		if err == nil {
			if strings.Contains(string(out), "DDR4") {
				return "DDR4"
			} else if strings.Contains(string(out), "DDR3") {
				return "DDR3"
			} else if strings.Contains(string(out), "DDR2") {
				return "DDR2"
			}
		}
	}
	return "Неизвестно"
}

// Вспомогательная функция для парсинга информации о модулях памяти
func parseMemoryModules(dmidecodeOutput string) string {
	var modules []string
	parts := strings.Split(dmidecodeOutput, "Memory Device")

	for _, part := range parts[1:] {
		size := "Неизвестно"
		speed := "Неизвестно"
		manufacturer := "Неизвестно"

		if strings.Contains(part, "Size:") {
			sizeLine := strings.Split(strings.Split(part, "Size:")[1], "\n")[0]
			size = strings.TrimSpace(sizeLine)
		}
		if strings.Contains(part, "Speed:") {
			speedLine := strings.Split(strings.Split(part, "Speed:")[1], "\n")[0]
			speed = strings.TrimSpace(speedLine)
		}
		if strings.Contains(part, "Manufacturer:") {
			manLine := strings.Split(strings.Split(part, "Manufacturer:")[1], "\n")[0]
			manufacturer = strings.TrimSpace(manLine)
		}

		if size != "No Module Installed" {
			modules = append(modules, fmt.Sprintf("%s %s %s", manufacturer, size, speed))
		}
	}

	return strings.Join(modules, "; ")
}
