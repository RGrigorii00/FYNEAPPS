package tabs

import (
	"context"
	"database/sql"
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
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
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

// Database connection and computer ID
var (
	db            *sql.DB
	dbEnabled     bool
	computerSaved bool // Флаг, что информация о компьютере уже сохранена
	computerID    int  // ID сохраненного компьютера в БД
)

func CreateHardwareTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Мониторинг системы", theme.Color(theme.ColorNameForeground))
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Добавляем переключатель для активации/деактивации сохранения в БД
	dbToggle := widget.NewCheck("Сохранять данные в БД", func(checked bool) {
		dbEnabled = checked
		if checked && db == nil {
			initDB()
		}
	})
	dbToggle.SetChecked(false)

	// Создаем контейнер для карточек (теперь 3 в ряд)
	cardsContainer := container.NewGridWithColumns(3)
	scrollContainer := container.NewVScroll(
		container.NewVBox(
			container.NewPadded(title),
			widget.NewSeparator(),
			container.NewHBox(layout.NewSpacer(), dbToggle, layout.NewSpacer()),
			widget.NewSeparator(),
			cardsContainer,
		),
	)
	scrollContainer.SetMinSize(fyne.NewSize(800, 600))

	// Канал для остановки обновлений
	stopChan := make(chan struct{})

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

		// Сохраняем данные в БД, если включено
		if dbEnabled {
			saveAllData()
		}
	}

	// Запускаем периодическое обновление
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				updateData()
			case <-stopChan:
				return
			}
		}
	}()

	// Кнопка ручного обновления
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), updateData)
	refreshBtn.Importance = widget.MediumImportance

	// Очистка при закрытии
	window.SetOnClosed(func() {
		close(stopChan) // Останавливаем горутину обновления
		if db != nil {
			db.Close()
		}
	})

	return container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), refreshBtn, layout.NewSpacer()),
		nil,
		nil,
		scrollContainer,
	)
}

// Инициализация БД
func initDB() {
	connStr := "user=user dbname=grafana_db password=user host=83.166.245.249 port=5432 sslmode=disable"
	var err error

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening database: %v", err)
		dbEnabled = false
		return
	}

	// Проверяем соединение
	err = db.Ping()
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		dbEnabled = false
		return
	}
}

// Сохраняет все данные в БД
func saveAllData() {
	if db == nil {
		return
	}

	// Получаем информацию о хосте
	hostInfo, err := host.Info()
	if err != nil {
		log.Printf("Error getting host info for DB: %v", err)
		return
	}

	// Получаем информацию о текущем пользователе
	currentUser, err := user.Current()
	if err != nil {
		log.Printf("Error getting current user for DB: %v", err)
		return
	}

	// Сохраняем основную информацию о компьютере
	err = saveComputerInfo(hostInfo, currentUser)
	if err != nil {
		log.Printf("Error saving computer info: %v", err)
	}

	// CPU информация
	cpuInfo, err := cpu.Info()
	if err != nil {
		log.Printf("Error getting CPU info for DB: %v", err)
		return
	}
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("Error getting CPU usage for DB: %v", err)
		return
	}

	// Сохраняем информацию о процессоре
	err = saveCPUInfo(cpuInfo, cpuUsage[0])
	if err != nil {
		log.Printf("Error saving CPU info: %v", err)
	}

	// Память
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory info for DB: %v", err)
		return
	}

	// Сохраняем информацию о памяти
	err = saveMemoryInfo(memInfo)
	if err != nil {
		log.Printf("Error saving memory info: %v", err)
	}

	// Диски
	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Printf("Error getting disk info for DB: %v", err)
		return
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

	// Сохраняем информацию о дисках
	err = saveDiskInfo(disks)
	if err != nil {
		log.Printf("Error saving disk info: %v", err)
	}

	// Сеть
	netInterfaces, err := net.IOCounters(true)
	if err != nil {
		log.Printf("Error getting network info for DB: %v", err)
		return
	}

	// Получаем дополнительную информацию о сетевых интерфейсах
	netStats, err := net.Interfaces()
	if err != nil {
		log.Printf("Error getting network stats for DB: %v", err)
	}

	// Сохраняем информацию о сети
	err = saveNetworkInfo(netInterfaces, netStats)
	if err != nil {
		log.Printf("Error saving network info: %v", err)
	}
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

// Сохраняет или обновляет информацию о компьютере
func saveComputerInfo(hostInfo *host.InfoStat, currentUser *user.User) error {
	if computerSaved {
		return nil // Уже сохранено, пропускаем
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Сначала пытаемся найти существующую запись
	err := db.QueryRowContext(ctx, `
        SELECT computer_id 
        FROM computers 
        WHERE host_name = $1 
        AND user_name = $2 
        AND os_name = $3 
        AND os_version = $4
        LIMIT 1`,
		hostInfo.Hostname,
		currentUser.Username,
		hostInfo.OS,
		hostInfo.PlatformVersion,
	).Scan(&computerID)

	if err == nil {
		// Запись найдена, используем существующий ID
		computerSaved = true
		return nil
	}

	// Если записи нет (ошибка не равна sql.ErrNoRows), возвращаем ошибку
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("ошибка поиска компьютера: %v", err)
	}

	// Создаем новую запись
	bootTime := time.Unix(int64(hostInfo.BootTime), 0)
	username := currentUser.Username

	err = db.QueryRowContext(ctx, `
        INSERT INTO computers (
            host_name, user_name, os_name, os_version, os_platform, 
            os_architecture, kernel_version, process_count, 
            boot_time, home_directory, gid, uid
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		hostInfo.Hostname,
		username,
		hostInfo.OS,
		hostInfo.PlatformVersion,
		hostInfo.Platform,
		hostInfo.KernelArch,
		hostInfo.KernelVersion,
		hostInfo.Procs,
		bootTime,
		currentUser.HomeDir,
		currentUser.Gid,
		currentUser.Uid,
	).Scan(&computerID)

	if err != nil {
		return fmt.Errorf("ошибка сохранения информации о компьютере: %v", err)
	}

	computerSaved = true
	return nil
}

// Сохраняет информацию о процессоре
func saveCPUInfo(cpuInfo []cpu.InfoStat, cpuUsage float64) error {
	if len(cpuInfo) == 0 || !computerSaved {
		return nil
	}

	_, err := db.Exec(`
        INSERT INTO processors (
            computer_id, model, manufacturer, architecture, 
            clock_speed, core_count, thread_count, usage_percent
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		computerID,
		cpuInfo[0].ModelName,
		cpuInfo[0].VendorID,
		runtime.GOARCH,
		float64(cpuInfo[0].Mhz)/1000, // GHz
		cpuInfo[0].Cores,
		runtime.NumCPU(), // threads
		cpuUsage,
	)

	return err
}

// Сохраняет информацию о памяти
func saveMemoryInfo(memInfo *mem.VirtualMemoryStat) error {
	if memInfo == nil || !computerSaved {
		return nil
	}

	totalGB := float64(memInfo.Total) / 1024 / 1024 / 1024
	usedGB := float64(memInfo.Used) / 1024 / 1024 / 1024
	freeGB := float64(memInfo.Free) / 1024 / 1024 / 1024

	_, err := db.Exec(`
        INSERT INTO memory (
            computer_id, total_memory_gb, used_memory_gb, 
            free_memory_gb, usage_percent, memory_type
        ) VALUES ($1, $2, $3, $4, $5, $6)`,
		computerID,
		totalGB,
		usedGB,
		freeGB,
		memInfo.UsedPercent,
		getMemoryType(),
	)

	return err
}

// Сохраняет информацию о дисках
func saveDiskInfo(disks []DiskInfo) error {
	if len(disks) == 0 || !computerSaved {
		return nil
	}

	for _, disk := range disks {
		totalGB := float64(disk.Total) / 1024 / 1024 / 1024
		usedGB := float64(disk.Used) / 1024 / 1024 / 1024
		freeGB := float64(disk.Free) / 1024 / 1024 / 1024

		_, err := db.Exec(`
            INSERT INTO disks (
                computer_id, drive_letter, total_space_gb, 
                used_space_gb, free_space_gb, usage_percent
            ) VALUES ($1, $2, $3, $4, $5, $6)`,
			computerID,
			disk.Name,
			totalGB,
			usedGB,
			freeGB,
			disk.Usage,
		)

		if err != nil {
			return err
		}
	}
	return nil
}

// Сохраняет информацию о сетевых адаптерах
func saveNetworkInfo(netInterfaces []net.IOCountersStat, netStats []net.InterfaceStat) error {
	if len(netInterfaces) == 0 || !computerSaved {
		return nil
	}

	now := time.Now()

	for _, iface := range netInterfaces {
		var uploadSpeed, downloadSpeed float64
		var macAddr string

		// Находим MAC-адрес
		for _, stat := range netStats {
			if stat.Name == iface.Name {
				macAddr = stat.HardwareAddr
				break
			}
		}

		// Рассчитываем скорости, если есть предыдущие данные
		if prevNetStats != nil && !prevNetTime.IsZero() {
			elapsed := now.Sub(prevNetTime).Seconds()

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
			uploadSpeed = float64(iface.BytesSent-prevBytesSent) / (1024 * 1024) / elapsed
			downloadSpeed = float64(iface.BytesRecv-prevBytesRecv) / (1024 * 1024) / elapsed
		}

		sentMB := float64(iface.BytesSent) / 1024 / 1024
		recvMB := float64(iface.BytesRecv) / 1024 / 1024

		_, err := db.Exec(`
            INSERT INTO network_adapters (
                computer_id, adapter_name, mac_address, upload_speed_mbps, 
                download_speed_mbps, sent_mb, received_mb, sent_packets, 
                received_packets, is_active
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			computerID,
			iface.Name,
			macAddr,
			uploadSpeed,
			downloadSpeed,
			sentMB,
			recvMB,
			iface.PacketsSent,
			iface.PacketsRecv,
			iface.BytesSent+iface.BytesRecv > 0,
		)

		if err != nil {
			return err
		}
	}

	// Сохраняем текущие значения для следующего расчета
	prevNetStats = netInterfaces
	prevNetTime = now

	return nil
}

// Модифицированная функция getSystemComponents для сохранения данных в БД
func getSystemComponents() ([]HardwareComponent, error) {
	var components []HardwareComponent

	// Получаем информацию о хосте
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting host info: %v", err)
	}

	// Получаем информацию о текущем пользователе
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("error getting current user: %v", err)
	}

	// Сохраняем основную информацию о компьютере
	if db != nil {
		err = saveComputerInfo(hostInfo, currentUser)
		if err != nil {
			log.Printf("Error saving computer info: %v", err)
		}
	}

	// CPU информация
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting CPU info: %v", err)
	}
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("error getting CPU usage: %v", err)
	}

	// Сохраняем информацию о процессоре
	if db != nil {
		err = saveCPUInfo(cpuInfo, cpuUsage[0])
		if err != nil {
			log.Printf("Error saving CPU info: %v", err)
		}
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

	// Сохраняем информацию о памяти
	if db != nil {
		err = saveMemoryInfo(memInfo)
		if err != nil {
			log.Printf("Error saving memory info: %v", err)
		}
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

	// Сохраняем информацию о дисках
	if db != nil {
		err = saveDiskInfo(disks)
		if err != nil {
			log.Printf("Error saving disk info: %v", err)
		}
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

	// Получаем дополнительную информацию о сетевых интерфейсах
	netStats, err := net.Interfaces()
	if err != nil {
		log.Printf("Ошибка получения информации о сетевых интерфейсах: %v", err)
	}

	// Сохраняем информацию о сети
	if db != nil {
		err = saveNetworkInfo(netInterfaces, netStats)
		if err != nil {
			log.Printf("Error saving network info: %v", err)
		}
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

	userDetails := map[string]string{
		"Имя хоста":      hostInfo.Hostname,
		"Время загрузки": fmt.Sprintf("%v", time.Unix(int64(hostInfo.BootTime), 0).Format("2006-01-02 15:04:05")),
		"Версия ядра":    hostInfo.KernelVersion,
		"ID хоста":       hostInfo.HostID,
	}

	// Добавляем информацию о пользователе
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

	components = append(components, HardwareComponent{
		ID:      "user",
		Name:    "Пользователь",
		Icon:    theme.AccountIcon(),
		Usage:   0,
		Details: userDetails,
	})

	return components, nil
}

// Остальные функции (createHardwareCard, getMemoryType, parseMemoryModules) остаются без изменений

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
