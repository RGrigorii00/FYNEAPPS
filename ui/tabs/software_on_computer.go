package tabs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows/registry"
)

type SystemSoftware struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
	Installed string `json:"installed"` // или string, в зависимости от parseWindowsInstallDate
}

var (
	softwareCache   []SystemSoftware
	softwareCacheMu sync.Mutex
	lastUpdateTime  time.Time
	cacheExpiry     = 5 * time.Minute
)

func CreateSoftwareTab(window fyne.Window) fyne.CanvasObject {
	title := canvas.NewText("Программы на компьютере (Загружает дольше обычного)", theme.ForegroundColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Задаем фиксированные ширины столбцов
	columnWidths := []float32{
		710, // Название
		210, // Версия
		170, // Издатель
		150, // Дата установки
	}

	// Создаем заголовки таблицы
	headerRow := container.NewHBox()
	headers := []string{"Название", "Издатель", "Версия", "Дата установки"}

	// Указываем индивидуальные отступы для заголовков
	headerLeftPaddings := []float32{
		310, // Название
		370, // Версия
		110, // Издатель
		110, // Дата установки
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
	softwareTable := widget.NewTable(
		func() (int, int) { return 0, len(columnWidths) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignLeading
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			// Заполнение будет в updateSoftware
		},
	)

	// Устанавливаем ширины столбцов
	for i, width := range columnWidths {
		softwareTable.SetColumnWidth(i, width)
	}

	// Элементы управления (как во вкладке процессов)
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по названию или издателю...")
	refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), nil)
	sortSelect := widget.NewSelect([]string{"Название", "Издатель", "Версия", "Дата установки"}, nil)
	sortSelect.SetSelected("Название")

	// Канал для остановки автообновления
	stopChan := make(chan struct{})

	// Функция получения данных с кэшированием
	getCachedSoftware := func() ([]SystemSoftware, error) {
		softwareCacheMu.Lock()
		defer softwareCacheMu.Unlock()

		if time.Since(lastUpdateTime) < cacheExpiry && len(softwareCache) > 0 {
			return softwareCache, nil
		}

		software, err := getInstalledSoftware()
		if err != nil {
			return nil, err
		}

		softwareCache = software
		lastUpdateTime = time.Now()
		return software, nil
	}

	// Функция обновления таблицы
	updateSoftware := func() {
		software, err := getCachedSoftware()
		if err != nil {
			log.Printf("Ошибка получения списка ПО: %v", err)
			return
		}

		filtered := filterSoftware(software, searchEntry.Text)
		sortSoftware(filtered, sortSelect.Selected)

		fyne.Do(func() {
			softwareTable.Length = func() (int, int) {
				return len(filtered), len(columnWidths)
			}
			softwareTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
				label := obj.(*widget.Label)
				if id.Row >= len(filtered) {
					label.SetText("")
					return
				}

				sw := filtered[id.Row]
				switch id.Col {
				case 0:
					label.SetText(sw.Name)
					label.Alignment = fyne.TextAlignLeading
					label.Wrapping = fyne.TextWrapWord
				case 1:
					label.SetText(sw.Version)
					label.Alignment = fyne.TextAlignLeading
					label.Wrapping = fyne.TextWrapWord
				case 2:
					label.SetText(sw.Publisher)
					label.Alignment = fyne.TextAlignLeading
					label.Wrapping = fyne.TextWrapWord
				case 3:
					label.SetText(sw.Installed)
					label.Alignment = fyne.TextAlignLeading
					label.Wrapping = fyne.TextWrapWord

				}
			}
			softwareTable.Refresh()
		})
	}

	// Автообновление (раз в 30 секунд)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				updateSoftware()
			case <-stopChan:
				return
			}
		}
	}()

	// Обработчики событий
	searchEntry.OnChanged = func(s string) { go updateSoftware() }
	refreshBtn.OnTapped = func() {
		softwareCacheMu.Lock()
		softwareCache = nil
		softwareCacheMu.Unlock()
		go updateSoftware()
	}
	sortSelect.OnChanged = func(s string) { go updateSoftware() }

	// Первоначальное обновление
	go updateSoftware()

	// Очистка при закрытии
	window.SetOnClosed(func() {
		close(stopChan)
	})

	// Компоновка элементов управления (как во вкладке процессов)
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

	// Главный контейнер
	mainContent := container.NewBorder(
		container.NewVBox(
			controls,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewBorder(
			headerRow,
			nil,
			nil,
			nil,
			container.NewScroll(softwareTable),
		),
	)

	return mainContent
}

func getInstalledSoftware() ([]SystemSoftware, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsSoftwareCombined()
	case "linux":
		return getLinuxSoftware()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func getWindowsSoftwareCombined() ([]SystemSoftware, error) {
	var result []SystemSoftware

	// Получаем программы из реестра
	regSoftware, err := getWindowsSoftwareFromRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to get software from registry: %v", err)
	}
	result = append(result, regSoftware...)

	// Красивый вывод с отступами
	saveToJSONWithSwappedFields(regSoftware)
	saveToPostgreSQL(regSoftware)

	// // Получаем программы из WMIC (MSI-установленные)
	// wmicSoftware, err := getWindowsSoftwareExtended()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get software from WMI: %v", err)
	// }
	// result = append(result, wmicSoftware...)

	// Удаляем дубликаты
	return removeSoftwareDuplicates(result), nil
}

func getWindowsSoftwareFromRegistry() ([]SystemSoftware, error) {
	keys := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	var softwareList []SystemSoftware

	for _, key := range keys {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		defer k.Close()

		subkeys, err := k.ReadSubKeyNames(-1)
		if err != nil {
			continue
		}

		for _, subkey := range subkeys {
			sk, err := registry.OpenKey(registry.LOCAL_MACHINE, key+`\`+subkey, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			name, _, _ := sk.GetStringValue("DisplayName")
			version, _, _ := sk.GetStringValue("Publisher")
			publisher, _, _ := sk.GetStringValue("DisplayVersion")
			installDate, _, _ := sk.GetStringValue("InstallDate")

			if name != "" {
				softwareList = append(softwareList, SystemSoftware{
					Name:      name,
					Publisher: publisher,
					Version:   version,
					Installed: parseWindowsInstallDate(installDate),
				})
			}

			sk.Close()
		}
	}

	return softwareList, nil
}

func getWindowsSoftwareExtended() ([]SystemSoftware, error) {
	// Вариант 1: Используем стандартный Win32_Product (только MSI-установленные)
	cmd := exec.Command("wmic", "product", "get", "name,vendor,version,installdate", "/format:csv")

	// Вариант 2: Альтернативный подход через Win32_AddRemovePrograms (если доступен)
	// cmd := exec.Command("wmic", "/namespace:\\root\\cimv2", "path", "Win32_AddRemovePrograms", "get", "DisplayName,Publisher,Version,InstallDate", "/format:csv")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("WMIC failed: %v", err)
	}

	lines := strings.Split(string(output), "\r\n")
	var softwareList []SystemSoftware

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 5 {
			installDate := parseWindowsInstallDate(parts[1])
			softwareList = append(softwareList, SystemSoftware{
				Name:      strings.TrimSpace(parts[2]),
				Version:   strings.TrimSpace(parts[3]),
				Publisher: strings.TrimSpace(parts[4]),
				Installed: installDate,
			})
		}
	}

	return softwareList, nil
}

// removeSoftwareDuplicates удаляет дубликаты программ из списка
func removeSoftwareDuplicates(software []SystemSoftware) []SystemSoftware {
	// Создаем map для отслеживания уникальных программ
	unique := make(map[string]SystemSoftware)

	for _, item := range software {
		if item.Name == "" {
			continue // Пропускаем записи без имени
		}

		// Создаем ключ на основе имени, версии и издателя
		key := fmt.Sprintf("%s|%s|%s", strings.ToLower(item.Name), strings.ToLower(item.Version), strings.ToLower(item.Publisher))

		// Если программа с таким ключом уже есть, выбираем более полную запись
		if existing, exists := unique[key]; exists {
			// Обновляем запись, если текущая имеет больше информации
			if item.Publisher != "" && existing.Publisher == "" {
				unique[key] = item
			} else if item.Version != "" && existing.Version == "" {
				unique[key] = item
			} else if item.Installed != "" && existing.Installed == "" {
				unique[key] = item
			}
		} else {
			unique[key] = item
		}
	}

	// Конвертируем map обратно в slice
	var result []SystemSoftware
	for _, item := range unique {
		result = append(result, item)
	}

	return result
}

func getLinuxSoftware() ([]SystemSoftware, error) {
	var softwareList []SystemSoftware

	// Получаем список пакетов через dpkg (Debian/Ubuntu)
	if _, err := os.Stat("/var/lib/dpkg/status"); err == nil {
		cmd := exec.Command("dpkg-query", "-W", "-f=${Package}\t${Version}\t${Maintainer}\t${Install-Date}\n")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("dpkg-query failed: %v", err)
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			parts := strings.Split(line, "\t")
			if len(parts) >= 4 {
				installDate := parseLinuxInstallDate(parts[3])
				softwareList = append(softwareList, SystemSoftware{
					Name:      strings.TrimSpace(parts[0]),
					Version:   strings.TrimSpace(parts[1]),
					Publisher: strings.TrimSpace(parts[2]),
					Installed: installDate,
				})
			}
		}
	}

	// Получаем список snap пакетов
	if _, err := exec.LookPath("snap"); err == nil {
		cmd := exec.Command("snap", "list")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("snap list failed: %v", err)
		} else {
			lines := strings.Split(string(output), "\n")
			for i, line := range lines {
				if i == 0 || strings.TrimSpace(line) == "" {
					continue
				}

				// Формат: Name Version Rev Tracking Publisher Notes
				fields := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), 6)
				if len(fields) >= 5 {
					softwareList = append(softwareList, SystemSoftware{
						Name:      fields[0],
						Version:   fields[1],
						Publisher: fields[4],
						Installed: "snap", // У snap нет даты установки в простом выводе
					})
				}
			}
		}
	}

	// Получаем список flatpak пакетов
	if _, err := exec.LookPath("flatpak"); err == nil {
		cmd := exec.Command("flatpak", "list", "--columns=application,origin,version,installation")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("flatpak list failed: %v", err)
		} else {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}

				fields := strings.Split(line, "\t")
				if len(fields) >= 3 {
					softwareList = append(softwareList, SystemSoftware{
						Name:      fields[0],
						Version:   fields[1],
						Publisher: fields[2],
						Installed: "flatpak", // Упрощенно
					})
				}
			}
		}
	}

	return softwareList, nil
}

func parseWindowsInstallDate(dateStr string) string {
	if len(dateStr) < 8 {
		return dateStr
	}
	return fmt.Sprintf("%s-%s-%s", dateStr[0:4], dateStr[4:6], dateStr[6:8])
}

func parseLinuxInstallDate(dateStr string) string {
	if len(dateStr) < 8 {
		return dateStr
	}
	// Формат даты в dpkg: 2023-04-20
	return dateStr
}

func filterSoftware(software []SystemSoftware, search string) []SystemSoftware {
	if search == "" {
		return software
	}

	var result []SystemSoftware
	search = strings.ToLower(search)
	for _, sw := range software {
		if strings.Contains(strings.ToLower(sw.Name), search) ||
			strings.Contains(strings.ToLower(sw.Publisher), search) {
			result = append(result, sw)
		}
	}
	return result
}

func sortSoftware(software []SystemSoftware, sortBy string) {
	switch sortBy {
	case "Название":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Name < software[j].Name
		})
	case "Издатель":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Version < software[j].Version
		})
	case "Версия":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Publisher < software[j].Publisher
		})
	case "Дата установки":
		sort.Slice(software, func(i, j int) bool {
			return software[i].Installed > software[j].Installed
		})
	}
}

// Временная структура только для JSON с перевернутыми полями
type jsonSystemSoftware struct {
	Name      string `json:"name"`
	Publisher string `json:"version"`   // Здесь Publisher сохраняется как version
	Version   string `json:"publisher"` // Здесь Version сохраняется как publisher
	Installed string `json:"installed"`
}

func saveToJSONWithSwappedFields(softwareList []SystemSoftware) error {
	// Конвертируем в временную структуру
	var jsonList []jsonSystemSoftware
	for _, s := range softwareList {
		jsonList = append(jsonList, jsonSystemSoftware{
			Name:      s.Name,
			Publisher: s.Publisher, // Publisher -> version в JSON
			Version:   s.Version,   // Version -> publisher в JSON
			Installed: s.Installed,
		})
	}

	// Сериализуем с отступами
	jsonData, err := json.MarshalIndent(jsonList, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %v", err)
	}
	fmt.Print("СОЗДАЛ ДЖСОН")

	// Записываем в файл
	if err := os.WriteFile("software.json", jsonData, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %v", err)
	}

	return nil
}

func saveToPostgreSQL(softwareList []SystemSoftware) error {
	hostName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("не удалось получить hostname: %v", err)
	}

	connString := "user=user dbname=grafana_db password=user host=83.166.245.249 port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %v", err)
	}
	defer db.Close()

	// Упрощаем настройки соединения
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Получаем существующие записи одним запросом
	existingRecords := make(map[string]bool)
	rows, err := db.Query("SELECT name, version, publisher FROM software_on_computer WHERE host_name = $1", hostName)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("ошибка получения существующих записей: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, version, publisher string
		if err := rows.Scan(&name, &version, &publisher); err != nil {
			return fmt.Errorf("ошибка чтения существующих записей: %v", err)
		}
		key := fmt.Sprintf("%s|%s|%s", name, version, publisher)
		existingRecords[key] = true
	}

	// Подготовка запроса на вставку
	stmt, err := db.Prepare(`
        INSERT INTO software_on_computer 
        (name, version, publisher, installed, metadata, host_name)
        VALUES ($1, $2, $3, $4, $5, $6)
    `)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %v", err)
	}
	defer stmt.Close()

	// Обработка программ
	var inserted, skipped int
	for i, s := range softwareList {
		// Пропускаем если имя совпадает с hostname
		if strings.EqualFold(s.Name, hostName) {
			skipped++
			continue
		}

		// Проверяем есть ли уже в базе
		key := fmt.Sprintf("%s|%s|%s", s.Name, s.Version, s.Publisher)
		if existingRecords[key] {
			skipped++
			continue
		}

		// Подготовка метаданных
		metadata, err := json.Marshal(s)
		if err != nil {
			return fmt.Errorf("ошибка сериализации: %v", err)
		}

		// Вставка с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err = stmt.ExecContext(ctx,
			nullIfEmpty(s.Name),
			nullIfEmpty(s.Version),
			nullIfEmpty(s.Publisher),
			nullIfEmpty(s.Installed),
			metadata,
			hostName,
		)

		if err != nil {
			return fmt.Errorf("ошибка вставки %s: %v", s.Name, err)
		}

		inserted++

		// Вывод прогресса
		if (i+1)%50 == 0 {
			fmt.Printf("Обработано %d из %d\n", i+1, len(softwareList))
		}
	}

	fmt.Printf("Готово. Вставлено: %d, Пропущено: %d\n", inserted, skipped)
	return nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
