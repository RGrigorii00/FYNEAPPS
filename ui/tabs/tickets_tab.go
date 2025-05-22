package tabs

import (
	"context"
	"database/sql"
	"fmt"
	"image/color"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "github.com/lib/pq"
)

type Ticket struct {
	ID           int
	Title        string
	Description  string
	UserID       string
	ComputerName string
	StatusID     int
	StatusName   string
	Cabinet      int
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}

type TicketsTab struct {
	window         fyne.Window
	content        fyne.CanvasObject
	cancelFunc     context.CancelFunc
	refreshChan    chan struct{}
	running        bool
	mutex          sync.RWMutex
	db             *sql.DB
	ticketsCache   []Ticket
	lastRefresh    time.Time
	isActive       bool
	sortField      string
	sortDescending bool
	ticketsList    *widget.List
	split          *container.Split
	statusSelect   *widget.Select
}

var (
	dbOnce       sync.Once
	dbMutex      sync.RWMutex
	statuses     = []string{"Новый", "В процессе", "Завершен"}
	statusValues = map[string]int{
		"Новый":      1,
		"В процессе": 2,
		"Завершен":   3,
	}
)

func showCustomDialog(window fyne.Window, title, message string, icon fyne.Resource) {
	fyne.Do(func() {
		dialog.ShowCustom(
			title,
			"OK",
			container.NewVBox(
				widget.NewIcon(icon),
				widget.NewLabel(message),
			),
			window,
		)
	})
}

func showCustomConfirmDialog(window fyne.Window, title, message string, icon fyne.Resource, confirmFunc func(bool)) {
	fyne.Do(func() {
		content := container.NewVBox(
			widget.NewIcon(icon),
			widget.NewLabel(message),
		)

		d := dialog.NewCustomConfirm(
			title,
			"Да",
			"Нет",
			content,
			confirmFunc,
			window,
		)
		d.Show()
	})
}

func getStatusColor(statusName string) color.Color {
	switch statusName {
	case "Завершен":
		return color.RGBA{R: 230, G: 255, B: 230, A: 255}
	case "В процессе":
		return color.RGBA{R: 255, G: 255, B: 200, A: 255}
	case "Новый":
		return color.RGBA{R: 255, G: 230, B: 230, A: 255}
	default:
		return color.White
	}
}

// Инициализация базы данных
func initDBT() (*sql.DB, error) {
	connStr := "user=user dbname=grafana_db password=user host=83.166.245.249 port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func CreateTicketsTab(window fyne.Window) (fyne.CanvasObject, func()) {
	tab := &TicketsTab{
		window:         window,
		refreshChan:    make(chan struct{}, 1),
		running:        true,
		sortField:      "created_at",
		sortDescending: true,
	}

	// В функции CreateTicketsTab:
	db, err := initDBT()
	if err != nil {
		showCustomDialog(window, "Ошибка", "Ошибка подключения к базе данных: "+err.Error(), theme.ErrorIcon())
		return widget.NewLabel("Ошибка подключения к БД"), func() {}
	}
	tab.db = db

	ctx, cancel := context.WithCancel(context.Background())
	tab.cancelFunc = cancel

	title := canvas.NewText("Управление тикетами", theme.ForegroundColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	ticketTitle := widget.NewEntry()
	ticketTitle.SetPlaceHolder("Заголовок тикета")
	ticketTitle.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Заголовок не может быть пустым")
		}
		return nil
	}

	ticketDesc := widget.NewMultiLineEntry()
	ticketDesc.SetPlaceHolder("Описание проблемы")
	ticketDesc.Wrapping = fyne.TextWrapWord
	ticketDesc.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Описание не может быть пустым")
		}
		return nil
	}

	userID := widget.NewEntry()
	userID.SetPlaceHolder("Имя пользоваля (из жизни)")
	userID.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Имя пользователя не может быть пусто")
		}
		return nil
	}

	computerName := widget.NewEntry()
	computerName.SetPlaceHolder("Имя компьютера")
	computerName.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("Имя компьютера не может быть пустым")
		}
		return nil
	}

	// Устанавливаем заранее заданное значение
	hn, err := os.Hostname()
	// if err != nil {
	// 	return nil
	// }
	computerName.SetText(hn)

	// Блокируем поле для ввода
	computerName.Disable()

	cabinet := widget.NewEntry()
	cabinet.SetPlaceHolder("Кабинет")
	cabinet.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		_, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("номер кабинета должен быть числом")
		}
		return nil
	}

	var selectedTicketID int = -1
	var lastSelectedID widget.ListItemID = -1

	ticketsList := widget.NewList(
		func() int {
			tab.mutex.RLock()
			defer tab.mutex.RUnlock()
			return len(tab.ticketsCache)
		},
		func() fyne.CanvasObject {
			card := canvas.NewRectangle(color.White)
			card.CornerRadius = 4
			card.StrokeWidth = 1
			card.StrokeColor = color.RGBA{R: 200, G: 200, B: 200, A: 255}

			selectionBorder := canvas.NewRectangle(color.Transparent)
			selectionBorder.CornerRadius = 4
			selectionBorder.StrokeWidth = 2
			selectionBorder.StrokeColor = theme.PrimaryColor()

			titleLabel := widget.NewLabel("")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}
			titleLabel.Truncation = fyne.TextTruncateEllipsis
			titleLabel.Wrapping = fyne.TextWrapWord

			descLabel := widget.NewLabel("")
			descLabel.Wrapping = fyne.TextWrapWord
			descLabel.TextStyle = fyne.TextStyle{Italic: true}

			metaContainer := container.NewGridWithColumns(2,
				widget.NewLabelWithStyle("Пользователь:", fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabel(""),
				widget.NewLabelWithStyle("Компьютер:", fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabel(""),
				widget.NewLabelWithStyle("Статус:", fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabel(""),
				widget.NewLabelWithStyle("Кабинет:", fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabel(""),
				widget.NewLabelWithStyle("Создан:", fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabel(""),
				widget.NewLabelWithStyle("Обновлен:", fyne.TextAlignLeading, fyne.TextStyle{}),
				widget.NewLabel(""),
			)

			content := container.NewVBox(
				container.NewPadded(titleLabel),
				container.NewPadded(descLabel),
				container.NewPadded(metaContainer),
			)

			return container.NewStack(
				card,
				container.NewPadded(content),
				selectionBorder,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			tab.mutex.RLock()
			defer tab.mutex.RUnlock()

			if i < 0 || i >= len(tab.ticketsCache) {
				return
			}

			stack := o.(*fyne.Container)
			card := stack.Objects[0].(*canvas.Rectangle)
			card.FillColor = getStatusColor(tab.ticketsCache[i].StatusName)

			contentContainer := stack.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
			titleLabel := contentContainer.Objects[0].(*fyne.Container).Objects[0].(*widget.Label)
			titleLabel.SetText(tab.ticketsCache[i].Title)

			descLabel := contentContainer.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
			descLabel.SetText(tab.ticketsCache[i].Description)

			metaContainer := contentContainer.Objects[2].(*fyne.Container).Objects[0].(*fyne.Container)
			metaContainer.Objects[1].(*widget.Label).SetText(tab.ticketsCache[i].UserID)
			metaContainer.Objects[3].(*widget.Label).SetText(tab.ticketsCache[i].ComputerName)

			statusLabel := metaContainer.Objects[5].(*widget.Label)
			statusLabel.SetText(tab.ticketsCache[i].StatusName)
			statusLabel.TextStyle.Bold = true

			metaContainer.Objects[7].(*widget.Label).SetText(fmt.Sprintf("%d", tab.ticketsCache[i].Cabinet))

			createdAtLabel := metaContainer.Objects[9].(*widget.Label)
			if tab.ticketsCache[i].CreatedAt != nil {
				createdAtLabel.SetText(tab.ticketsCache[i].CreatedAt.Format("02.01.2006 15:04"))
			} else {
				createdAtLabel.SetText("не указана")
			}

			updatedAtLabel := metaContainer.Objects[11].(*widget.Label)
			if tab.ticketsCache[i].UpdatedAt != nil {
				updatedAtLabel.SetText(tab.ticketsCache[i].UpdatedAt.Format("02.01.2006 15:04"))
			} else {
				updatedAtLabel.SetText("не обновлялся")
			}

			border := stack.Objects[2].(*canvas.Rectangle)
			if i == lastSelectedID {
				border.StrokeColor = theme.PrimaryColor()
			} else {
				border.StrokeColor = color.Transparent
			}
		},
	)

	tab.ticketsList = ticketsList

	statusSelect := widget.NewSelect(statuses, nil)
	statusSelect.PlaceHolder = "Выберите статус"
	tab.statusSelect = statusSelect

	updateStatusBtn := widget.NewButtonWithIcon("Обновить статус", theme.ViewRefreshIcon(), func() {
		if selectedTicketID == -1 {
			showCustomDialog(window, "Ошибка", "Выберите тикет для изменения статуса", theme.WarningIcon())
			return
		}

		if statusSelect.Selected == "" {
			showCustomDialog(window, "Ошибка", "Выберите новый статус", theme.WarningIcon())
			return
		}

		statusID := statusValues[statusSelect.Selected]
		if err := updateTicketStatus(tab.db, selectedTicketID, statusID); err != nil {
			showCustomDialog(window, "Ошибка", "Не удалось обновить статус: "+err.Error(), theme.ErrorIcon())
			return
		}

		showCustomDialog(window, "Успех", "Статус тикета успешно обновлен", theme.ConfirmIcon())
		tab.refreshChan <- struct{}{}
	})

	ticketsList.OnSelected = func(id widget.ListItemID) {
		tab.mutex.RLock()
		defer tab.mutex.RUnlock()

		if id == lastSelectedID {
			selectedTicketID = -1
			lastSelectedID = -1
			ticketsList.UnselectAll()

			ticketTitle.SetText("")
			ticketDesc.SetText("")
			userID.SetText("")
			computerName.SetText("")
			cabinet.SetText("")
			statusSelect.SetSelected("")
			return
		}

		if id >= 0 && id < len(tab.ticketsCache) {
			selectedTicketID = tab.ticketsCache[id].ID
			lastSelectedID = id

			ticketTitle.SetText(tab.ticketsCache[id].Title)
			ticketDesc.SetText(tab.ticketsCache[id].Description)
			userID.SetText(tab.ticketsCache[id].UserID)
			computerName.SetText(tab.ticketsCache[id].ComputerName)
			cabinet.SetText(fmt.Sprintf("%d", tab.ticketsCache[id].Cabinet))
			statusSelect.SetSelected(tab.ticketsCache[id].StatusName)
		}
		fyne.Do(func() {
			ticketsList.Refresh()
		})
	}

	refreshList := func() {
		if !tab.running {
			return
		}

		if time.Since(tab.lastRefresh) < 500*time.Millisecond {
			return
		}

		tickets, err := getTickets(tab.db, tab.sortField, tab.sortDescending)
		if err != nil {
			log.Printf("Ошибка получения тикетов: %v", err)
			return
		}

		tab.mutex.Lock()
		tab.ticketsCache = tickets
		tab.lastRefresh = time.Now()
		tab.mutex.Unlock()

		fyne.Do(func() {
			ticketsList.Refresh()
		})
	}

	clearBtn := widget.NewButtonWithIcon("Убрать выделение с тикета", theme.DeleteIcon(), func() {
		selectedTicketID = -1
		lastSelectedID = -1
		ticketsList.UnselectAll()
		ticketTitle.SetText("")
		ticketDesc.SetText("")
		userID.SetText("")
		computerName.SetText(hn)
		cabinet.SetText("")
		statusSelect.SetSelected("")
	})

	createBtn := widget.NewButtonWithIcon("Создать", theme.ContentAddIcon(), func() {
		if err := ticketTitle.Validate(); err != nil {
			showCustomDialog(window, "Ошибка", err.Error(), theme.WarningIcon())
			return
		}

		if err := cabinet.Validator(cabinet.Text); err != nil {
			showCustomDialog(window, "Ошибка", err.Error(), theme.WarningIcon())
			return
		}

		cabinetValue := 0
		if cabinet.Text != "" {
			cabinetValue, _ = strconv.Atoi(cabinet.Text)
		}

		now := time.Now()
		ticket := Ticket{
			Title:        ticketTitle.Text,
			Description:  ticketDesc.Text,
			UserID:       userID.Text,
			ComputerName: computerName.Text,
			StatusID:     1,
			Cabinet:      cabinetValue,
			CreatedAt:    &now,
			UpdatedAt:    nil,
		}

		if err := addTicket(tab.db, ticket); err != nil {
			showCustomDialog(window, "Ошибка", "Не удалось создать тикет: "+err.Error(), theme.ErrorIcon())
			return
		}

		ticketTitle.SetText("")
		ticketDesc.SetText("")
		userID.SetText("")
		computerName.SetText("")
		cabinet.SetText("")
		showCustomDialog(window, "Успех", "Тикет успешно создан", theme.ConfirmIcon())
		tab.refreshChan <- struct{}{}
	})

	updateBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		if selectedTicketID == -1 {
			showCustomDialog(window, "Ошибка", "Выберите тикет для обновления", theme.WarningIcon())
			return
		}

		cabinetValue := 0
		if cabinet.Text != "" {
			cabinetValue, _ = strconv.Atoi(cabinet.Text)
		}

		ticket := Ticket{
			ID:           selectedTicketID,
			Title:        ticketTitle.Text,
			Description:  ticketDesc.Text,
			UserID:       userID.Text,
			ComputerName: computerName.Text,
			Cabinet:      cabinetValue,
		}

		if err := updateTicket(tab.db, ticket); err != nil {
			showCustomDialog(window, "Ошибка", "Не удалось обновить тикет: "+err.Error(), theme.ErrorIcon())
			return
		}

		showCustomDialog(window, "Успех", "Тикет успешно обновлен", theme.ConfirmIcon())
		tab.refreshChan <- struct{}{}
	})

	deleteBtn := widget.NewButtonWithIcon("Удалить", theme.DeleteIcon(), func() {
		if selectedTicketID == -1 {
			showCustomDialog(window, "Ошибка", "Выберите тикет для удаления", theme.WarningIcon())
			return
		}

		var ticketTitle string
		tab.mutex.RLock()
		for _, t := range tab.ticketsCache {
			if t.ID == selectedTicketID {
				ticketTitle = t.Title
				break
			}
		}
		tab.mutex.RUnlock()

		showCustomConfirmDialog(
			window,
			"Подтверждение удаления",
			fmt.Sprintf("Вы уверены, что хотите удалить тикет '%s'?\nЭто действие нельзя отменить.", ticketTitle),
			theme.WarningIcon(),
			func(ok bool) {
				if ok {
					if err := deleteTicket(tab.db, selectedTicketID); err != nil {
						showCustomDialog(window, "Ошибка", "Не удалось удалить тикет: "+err.Error(), theme.ErrorIcon())
						return
					}
					selectedTicketID = -1
					lastSelectedID = -1
					showCustomDialog(window, "Успех", "Тикет успешно удален", theme.ConfirmIcon())
					tab.refreshChan <- struct{}{}
				}
			},
		)
	})

	sortOptions := []string{"ID", "Заголовок", "Статус", "Кабинет", "Дата создания", "Дата обновления"}
	sortSelect := widget.NewSelect(sortOptions, func(selected string) {
		switch selected {
		case "ID":
			tab.sortField = "id"
		case "Заголовок":
			tab.sortField = "title"
		case "Статус":
			tab.sortField = "status_id"
		case "Кабинет":
			tab.sortField = "cabinet"
		case "Дата создания":
			tab.sortField = "created_at"
		case "Дата обновления":
			tab.sortField = "update_at"
		}
		tab.refreshChan <- struct{}{}
	})
	sortSelect.SetSelected("Дата создания")

	var sortDirectionBtn *widget.Button
	sortDirectionBtn = widget.NewButton("По возрастанию", func() {
		tab.sortDescending = !tab.sortDescending

		if tab.sortDescending {
			sortDirectionBtn.SetText("По убыванию")
		} else {
			sortDirectionBtn.SetText("По возрастанию")
		}

		tab.refreshChan <- struct{}{}
	})

	sortContainer := container.NewHBox(
		widget.NewLabel("Сортировка:"),
		sortSelect,
		sortDirectionBtn,
	)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		refreshList()

		for {
			select {
			case <-ticker.C:
				refreshList()
			case <-tab.refreshChan:
				refreshList()
			case <-ctx.Done():
				return
			}
		}
	}()

	form := container.NewVBox(
		widget.NewLabelWithStyle("Создать/редактировать тикет", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Заголовок", ticketTitle),
			widget.NewFormItem("Описание", ticketDesc),
			widget.NewFormItem("ФИО пользователя", userID),
			widget.NewFormItem("Имя компьютера", computerName),
			widget.NewFormItem("Кабинет", cabinet),
		),
		container.NewHBox(
			createBtn,
			updateBtn,
			clearBtn,
		),
	)

	statusForm := container.NewVBox(
		widget.NewLabelWithStyle("Изменить статус тикета", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(
			nil,
			container.NewHBox(
				widget.NewLabel(""),
				updateStatusBtn,
				widget.NewLabel(""),
			),
			nil,
			nil,
			statusSelect,
		),
	)

	leftPanel := container.NewVBox(
		form,
		widget.NewSeparator(),
		statusForm,
	)

	split := container.NewHSplit(
		container.NewPadded(leftPanel),
		container.NewPadded(ticketsList),
	)
	tab.split = split

	split.SetOffset(0.4)

	tab.content = container.NewBorder(
		container.NewVBox(
			container.NewCenter(title),
			widget.NewSeparator(),
			sortContainer,
			widget.NewSeparator(),
		),
		container.NewHBox(
			widget.NewLabel(""),
			container.NewCenter(deleteBtn),
			widget.NewLabel(""),
		),
		nil,
		nil,
		split,
	)

	cleanup := func() {
		tab.mutex.Lock()
		defer tab.mutex.Unlock()

		if !tab.running {
			return
		}

		tab.running = false
		if tab.cancelFunc != nil {
			tab.cancelFunc()
		}
		close(tab.refreshChan)

		if tab.db != nil {
			tab.db.Close()
		}
	}

	return tab.content, cleanup
}

func getTickets(db *sql.DB, sortField string, sortDescending bool) ([]Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sortDirection := "DESC"
	if !sortDescending {
		sortDirection = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT t.id, t.title, t.description, t.user_id,
		       t.computer_name, t.status_id, ts.name, t.cabinet, 
		       t.created_at, t.update_at
		FROM tickets t
		JOIN tickets_statuses ts ON t.status_id = ts.id
		ORDER BY %s %s`, sortField, sortDirection)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(
			&ticket.ID,
			&ticket.Title,
			&ticket.Description,
			&ticket.UserID,
			&ticket.ComputerName,
			&ticket.StatusID,
			&ticket.StatusName,
			&ticket.Cabinet,
			&createdAt,
			&updatedAt); err != nil {
			return nil, err
		}

		if createdAt.Valid {
			ticket.CreatedAt = &createdAt.Time
		}
		if updatedAt.Valid {
			ticket.UpdatedAt = &updatedAt.Time
		}

		tickets = append(tickets, ticket)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func addTicket(db *sql.DB, ticket Ticket) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO tickets 
		(title, description, user_id, computer_name, status_id, cabinet, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.ExecContext(ctx, query,
		ticket.Title,
		ticket.Description,
		ticket.UserID,
		ticket.ComputerName,
		ticket.StatusID,
		ticket.Cabinet,
		ticket.CreatedAt)
	return err
}

func updateTicket(db *sql.DB, ticket Ticket) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	now := time.Now()
	query := `
		UPDATE tickets 
		SET title = $1, description = $2, user_id = $3, 
		    computer_name = $4, cabinet = $5, update_at = $6 
		WHERE id = $7`
	_, err := db.ExecContext(ctx, query,
		ticket.Title,
		ticket.Description,
		ticket.UserID,
		ticket.ComputerName,
		ticket.Cabinet,
		now,
		ticket.ID)
	return err
}

func updateTicketStatus(db *sql.DB, ticketID, statusID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	now := time.Now()
	query := `
		UPDATE tickets 
		SET status_id = $1, update_at = $2 
		WHERE id = $3`
	_, err := db.ExecContext(ctx, query,
		statusID,
		now,
		ticketID)
	return err
}

func deleteTicket(db *sql.DB, id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM tickets WHERE id = $1`
	_, err := db.ExecContext(ctx, query, id)
	return err
}
