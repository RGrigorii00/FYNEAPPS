package tabs

import (
	"context"
	"database/sql"
	"fmt"
	"image/color"
	"log"
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
}

type TicketsTab struct {
	window       fyne.Window
	content      fyne.CanvasObject
	cancelFunc   context.CancelFunc
	refreshChan  chan struct{}
	running      bool
	mutex        sync.RWMutex
	dbt          *sql.DB
	ticketsCache []Ticket
	lastRefresh  time.Time
	isActive     bool // Добавляем флаг активности вкладки
}

var (
	dbtOnce  sync.Once
	dbtMutex sync.RWMutex
)

// showCustomDialog создает кастомное всплывающее окно
func (t *TicketsTab) showCustomDialog(title, message string, icon fyne.Resource) {
	dialog.ShowCustom(
		title,
		"OK",
		container.NewVBox(
			widget.NewIcon(icon),
			widget.NewLabel(message),
		),
		t.window,
	)
}

// getStatusColor возвращает цвет в зависимости от статуса тикета
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

// getDB инициализирует и возвращает соединение с БД
func (t *TicketsTab) getDB() (*sql.DB, error) {
	var initErr error
	dbtOnce.Do(func() {
		connStr := "user=user dbname=grafana_db password=user host=83.166.245.249 port=5432 sslmode=disable"
		t.dbt, initErr = sql.Open("postgres", connStr)
		if initErr != nil {
			return
		}

		t.dbt.SetMaxOpenConns(25)
		t.dbt.SetMaxIdleConns(5)
		t.dbt.SetConnMaxLifetime(5 * time.Minute)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		initErr = t.dbt.PingContext(ctx)
	})

	return t.dbt, initErr
}

// CreateTicketsTab создает и возвращает вкладку управления тикетами
func CreateTicketsTab(window fyne.Window) (fyne.CanvasObject, func()) {
	tab := &TicketsTab{
		window:      window,
		refreshChan: make(chan struct{}, 1),
		running:     true,
	}

	// Инициализация подключения к БД
	if _, err := tab.getDB(); err != nil {
		tab.showCustomDialog("Ошибка", "Ошибка подключения к базе данных: "+err.Error(), theme.ErrorIcon())
		return widget.NewLabel("Ошибка подключения к БД"), func() {}
	}

	// Создаем контекст для управления горутинами
	ctx, cancel := context.WithCancel(context.Background())
	tab.cancelFunc = cancel

	// UI элементы
	title := widget.NewLabel("Управление тикетами (PostgreSQL)")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	ticketTitle := widget.NewEntry()
	ticketTitle.SetPlaceHolder("Заголовок тикета")
	ticketTitle.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("заголовок не может быть пустым")
		}
		return nil
	}

	ticketDesc := widget.NewMultiLineEntry()
	ticketDesc.SetPlaceHolder("Описание проблемы")
	ticketDesc.Wrapping = fyne.TextWrapWord

	userID := widget.NewEntry()
	userID.SetPlaceHolder("ID пользователя")

	computerName := widget.NewEntry()
	computerName.SetPlaceHolder("Имя компьютера")

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

	// Создаем стилизованный список тикетов
	ticketsList := widget.NewList(
		func() int {
			tab.mutex.RLock()
			defer tab.mutex.RUnlock()
			return len(tab.ticketsCache)
		},
		func() fyne.CanvasObject {
			card := canvas.NewRectangle(color.White)
			card.CornerRadius = 8
			card.StrokeWidth = 1
			card.StrokeColor = color.RGBA{R: 200, G: 200, B: 200, A: 255}

			selectionBorder := canvas.NewRectangle(color.Transparent)
			selectionBorder.CornerRadius = 8
			selectionBorder.StrokeWidth = 4
			selectionBorder.StrokeColor = theme.PrimaryColor()

			titleLabel := widget.NewLabel("Template")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}
			titleLabel.Truncation = fyne.TextTruncateEllipsis

			descLabel := widget.NewLabel("Template description")
			descLabel.Wrapping = fyne.TextWrapWord
			descLabel.TextStyle = fyne.TextStyle{Italic: true}

			metaContainer := container.NewGridWithColumns(2,
				widget.NewLabel("User ID:"),
				widget.NewLabel("0"),
				widget.NewLabel("Computer:"),
				widget.NewLabel("Template"),
				widget.NewLabel("Status:"),
				widget.NewLabel("Template"),
				widget.NewLabel("Cabinet:"),
				widget.NewLabel("0"),
			)

			content := container.NewVBox(
				titleLabel,
				descLabel,
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
			titleLabel := contentContainer.Objects[0].(*widget.Label)
			titleLabel.SetText(tab.ticketsCache[i].Title)

			descLabel := contentContainer.Objects[1].(*widget.Label)
			descLabel.SetText(tab.ticketsCache[i].Description)

			metaContainer := contentContainer.Objects[2].(*fyne.Container).Objects[0].(*fyne.Container)
			metaContainer.Objects[1].(*widget.Label).SetText(tab.ticketsCache[i].UserID)
			metaContainer.Objects[3].(*widget.Label).SetText(tab.ticketsCache[i].ComputerName)

			statusLabel := metaContainer.Objects[5].(*widget.Label)
			statusLabel.SetText(tab.ticketsCache[i].StatusName)
			statusLabel.TextStyle.Bold = true

			metaContainer.Objects[7].(*widget.Label).SetText(fmt.Sprintf("%d", tab.ticketsCache[i].Cabinet))

			border := stack.Objects[2].(*canvas.Rectangle)
			if i == lastSelectedID {
				border.StrokeColor = theme.PrimaryColor()
			} else {
				border.StrokeColor = color.Transparent
			}
		},
	)

	// Обработчик выбора элемента
	ticketsList.OnSelected = func(id widget.ListItemID) {
		tab.mutex.RLock()
		defer tab.mutex.RUnlock()

		if id >= 0 && id < len(tab.ticketsCache) {
			selectedTicketID = tab.ticketsCache[id].ID
			lastSelectedID = id

			ticketTitle.SetText(tab.ticketsCache[id].Title)
			ticketDesc.SetText(tab.ticketsCache[id].Description)
			userID.SetText(tab.ticketsCache[id].UserID)
			computerName.SetText(tab.ticketsCache[id].ComputerName)
			cabinet.SetText(fmt.Sprintf("%d", tab.ticketsCache[id].Cabinet))
			ticketsList.Refresh()
		}
	}

	// Функция обновления списка тикетов
	refreshList := func() {
		if !tab.running {
			return
		}

		if time.Since(tab.lastRefresh) < 500*time.Millisecond {
			return
		}

		tickets, err := tab.getTickets()
		if err != nil {
			log.Printf("Ошибка получения тикетов: %v", err)
			return
		}

		tab.mutex.Lock()
		tab.ticketsCache = tickets
		tab.lastRefresh = time.Now()
		tab.mutex.Unlock()

		window.Canvas().Refresh(ticketsList)
	}

	// Кнопка создания тикета
	createBtn := widget.NewButtonWithIcon("Создать", theme.ContentAddIcon(), func() {
		if err := ticketTitle.Validate(); err != nil {
			tab.showCustomDialog("Ошибка", err.Error(), theme.WarningIcon())
			return
		}

		if err := cabinet.Validator(cabinet.Text); err != nil {
			tab.showCustomDialog("Ошибка", err.Error(), theme.WarningIcon())
			return
		}

		cabinetValue := 0
		if cabinet.Text != "" {
			cabinetValue, _ = strconv.Atoi(cabinet.Text)
		}

		ticket := Ticket{
			Title:        ticketTitle.Text,
			Description:  ticketDesc.Text,
			UserID:       userID.Text,
			ComputerName: computerName.Text,
			StatusID:     1,
			Cabinet:      cabinetValue,
		}

		if err := tab.addTicket(ticket); err != nil {
			tab.showCustomDialog("Ошибка", "Не удалось создать тикет: "+err.Error(), theme.ErrorIcon())
			return
		}

		ticketTitle.SetText("")
		ticketDesc.SetText("")
		userID.SetText("")
		computerName.SetText("")
		cabinet.SetText("")

		tab.showCustomDialog("Успех", "Тикет успешно создан", theme.ConfirmIcon())
		tab.refreshChan <- struct{}{}
	})

	// Кнопка обновления тикета
	updateBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
		if selectedTicketID == -1 {
			tab.showCustomDialog("Ошибка", "Выберите тикет для обновления", theme.WarningIcon())
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

		if err := tab.updateTicket(ticket); err != nil {
			tab.showCustomDialog("Ошибка", "Не удалось обновить тикет: "+err.Error(), theme.ErrorIcon())
			return
		}

		tab.showCustomDialog("Успех", "Тикет успешно обновлен", theme.ConfirmIcon())
		tab.refreshChan <- struct{}{}
	})

	// Кнопка удаления тикета
	deleteBtn := widget.NewButtonWithIcon("Удалить", theme.DeleteIcon(), func() {
		if selectedTicketID == -1 {
			tab.showCustomDialog("Ошибка", "Выберите тикет для удаления", theme.WarningIcon())
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

		confirm := dialog.NewConfirm(
			"Подтверждение",
			fmt.Sprintf("Вы уверены, что хотите удалить тикет '%s'?", ticketTitle),
			func(ok bool) {
				if ok {
					if err := tab.deleteTicket(selectedTicketID); err != nil {
						tab.showCustomDialog("Ошибка", "Не удалось удалить тикет: "+err.Error(), theme.ErrorIcon())
						return
					}
					selectedTicketID = -1
					lastSelectedID = -1
					tab.showCustomDialog("Успех", "Тикет успешно удален", theme.ConfirmIcon())
					tab.refreshChan <- struct{}{}
				}
			},
			window,
		)
		confirm.Show()
	})

	// Запускаем горутину для периодического обновления
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Первоначальная загрузка
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

	// Компоновка интерфейса
	form := container.NewVBox(
		widget.NewLabelWithStyle("Создать/редактировать тикет", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Заголовок", ticketTitle),
			widget.NewFormItem("Описание", ticketDesc),
			widget.NewFormItem("ID пользователя", userID),
			widget.NewFormItem("Компьютер", computerName),
			widget.NewFormItem("Кабинет", cabinet),
		),
		container.NewHBox(
			createBtn,
			updateBtn,
		),
	)

	tab.content = container.NewBorder(
		container.NewCenter(title),
		container.NewHBox(
			widget.NewLabel(""),
			container.NewCenter(deleteBtn),
			widget.NewLabel(""),
		),
		nil,
		nil,
		container.NewHSplit(
			container.NewPadded(form),
			container.NewPadded(ticketsList),
		),
	)

	// Функция очистки для вызова при закрытии вкладки
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

		if tab.dbt != nil {
			tab.dbt.Close()
		}
	}

	return tab.content, cleanup
}

// Методы работы с БД
func (t *TicketsTab) getTickets() ([]Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT t.id, t.title, t.description, t.user_id, 
		       t.computer_name, t.status_id, ts.name, t.cabinet 
		FROM tickets t
		JOIN tickets_statuses ts ON t.status_id = ts.id
		ORDER BY t.id DESC`
	rows, err := t.dbt.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.UserID,
			&t.ComputerName,
			&t.StatusID,
			&t.StatusName,
			&t.Cabinet); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (t *TicketsTab) addTicket(ticket Ticket) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO tickets 
		(title, description, user_id, computer_name, status_id, cabinet) 
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := t.dbt.ExecContext(ctx, query,
		ticket.Title,
		ticket.Description,
		ticket.UserID,
		ticket.ComputerName,
		ticket.StatusID,
		ticket.Cabinet)
	return err
}

func (t *TicketsTab) updateTicket(ticket Ticket) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		UPDATE tickets 
		SET title = $1, description = $2, user_id = $3, 
		    computer_name = $4, cabinet = $5 
		WHERE id = $6`
	_, err := t.dbt.ExecContext(ctx, query,
		ticket.Title,
		ticket.Description,
		ticket.UserID,
		ticket.ComputerName,
		ticket.Cabinet,
		ticket.ID)
	return err
}

func (t *TicketsTab) deleteTicket(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM tickets WHERE id = $1`
	_, err := t.dbt.ExecContext(ctx, query, id)
	return err
}

// cleanup освобождает ресурсы при закрытии вкладки
func (t *TicketsTab) cleanup() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.running {
		return
	}

	t.running = false
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	close(t.refreshChan)
}
