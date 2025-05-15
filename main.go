package main

import (
	"FYNEAPPS/resources"
	"FYNEAPPS/ui"
	setting "FYNEAPPS/ui/setting_tab"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"github.com/getlantern/systray"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var (
	db          *sql.DB
	myApp       fyne.App
	mainWindow  fyne.Window
	loginWindow fyne.Window
	currentUser *User
)

const (
	dbHost      = "83.166.245.249"
	dbPort      = 5432
	dbUser      = "user"
	dbPassword  = "user"
	dbName      = "grafana_db"
	sessionFile = "session.json" // Файл для хранения сессии
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
	LoggedIn  bool      `json:"logged_in"`
}

type SessionData struct {
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func initDB() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

func authenticate(username, password string) (*User, error) {
	query := `SELECT id, username, full_name FROM users WHERE username = $1 AND password = $2`

	var user User
	err := db.QueryRow(query, username, password).Scan(&user.ID, &user.Username, &user.FullName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка аутентификации: %v", err)
	}

	user.SessionID = uuid.New().String()
	user.ExpiresAt = time.Now().Add(24 * time.Hour)
	user.LoggedIn = true

	_, err = db.Exec(
		"INSERT INTO user_sessions (user_id, session_id, expires_at) VALUES ($1, $2, $3)",
		user.ID, user.SessionID, user.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сессии: %v", err)
	}

	// Сохраняем сессию в JSON
	if err := saveSessionToFile(user.SessionID, user.ExpiresAt); err != nil {
		return nil, fmt.Errorf("ошибка сохранения сессии: %v", err)
	}

	return &user, nil
}

func loadSessionFromDB(sessionID string) (*User, error) {
	query := `
		SELECT u.id, u.username, u.full_name, s.expires_at 
		FROM users u JOIN user_sessions s ON u.id = s.user_id
		WHERE s.session_id = $1 AND s.expires_at > NOW()`

	var user User
	err := db.QueryRow(query, sessionID).Scan(&user.ID, &user.Username, &user.FullName, &user.ExpiresAt)
	if err != nil {
		return nil, err
	}

	user.SessionID = sessionID
	user.LoggedIn = true
	return &user, nil
}

func saveSessionToFile(sessionID string, expiresAt time.Time) error {
	data := SessionData{
		SessionID: sessionID,
		ExpiresAt: expiresAt,
	}

	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка кодирования сессии: %v", err)
	}

	if err := os.WriteFile(sessionFile, file, 0600); err != nil {
		return fmt.Errorf("ошибка записи файла сессии: %v", err)
	}

	return nil
}

func loadSessionFromFile() (*SessionData, error) {
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("файл сессии не найден")
	}

	file, err := os.ReadFile(sessionFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла сессии: %v", err)
	}

	var session SessionData
	if err := json.Unmarshal(file, &session); err != nil {
		return nil, fmt.Errorf("ошибка декодирования сессии: %v", err)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("сессия истекла")
	}

	return &session, nil
}

func clearSessionFile() error {
	if err := os.Remove(sessionFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления файла сессии: %v", err)
	}
	return nil
}

func loadSavedSession() (*User, error) {
	session, err := loadSessionFromFile()
	if err != nil {
		return nil, err
	}

	return loadSessionFromDB(session.SessionID)
}

func closeWindowSafely(w *fyne.Window) {
	if w != nil && *w != nil {
		(*w).Close()
		*w = nil
	}
}

func logout() {
	log.Println("Начало выхода из системы")

	if currentUser != nil && currentUser.SessionID != "" {
		if _, err := db.Exec("DELETE FROM user_sessions WHERE session_id = $1", currentUser.SessionID); err != nil {
			log.Printf("Ошибка удаления сессии: %v", err)
		}
	}

	if err := clearSessionFile(); err != nil {
		log.Printf("Ошибка очистки сессии: %v", err)
	}

	currentUser = nil

	fyne.Do(func() {
		closeWindowSafely(&mainWindow)
		showLoginWindow()
	})

	log.Println("Выход из системы завершен")
}

func showLoginWindow() {
	if loginWindow != nil {
		loginWindow.Show()
		loginWindow.RequestFocus()
		return
	}

	loginWindow = ui.CreateLoginWindow(myApp, func(username, password string) bool {
		if username == "" || password == "" {
			dialog.ShowError(fmt.Errorf("Введите логин и пароль"), loginWindow)
			return false
		}

		user, err := authenticate(username, password)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Ошибка входа: %v", err), loginWindow)
			return false
		}

		if user != nil {
			currentUser = user
			closeWindowSafely(&loginWindow)
			showMainWindow()
			return true
		}

		dialog.ShowError(fmt.Errorf("Неверные учетные данные"), loginWindow)
		return false
	})

	loginWindow.SetCloseIntercept(func() {
		loginWindow.Hide()
	})

	loginWindow.Show()
	loginWindow.CenterOnScreen()
}

func showMainWindow() {
	if mainWindow != nil {
		mainWindow.Close()
	}

	mainWindow = myApp.NewWindow("ПГАТУ Инфраструктура")
	mainWindow.CenterOnScreen()

	if icon := resources.ResourcePgatulogosmallPng; icon != nil {
		mainWindow.SetIcon(icon)
	}

	appSettings := setting.LoadSettings(myApp, mainWindow)
	mainWindow.Resize(fyne.NewSize(float32(appSettings.Width), float32(appSettings.Height)))

	mainWindow.SetCloseIntercept(func() {
		fyne.Do(func() {
			mainWindow.Hide()
		})
	})

	appTabs := ui.CreateAppTabs(myApp, mainWindow)
	mainWindow.SetContent(appTabs)

	mainWindow.SetOnClosed(func() {
		mainWindow = nil
	})

	mainWindow.Show()
}

func setupTray() {
	if iconData := resources.ResourcePgatulogosmallicoIco.StaticContent; len(iconData) > 0 {
		systray.SetIcon(iconData)
	}
	systray.SetTooltip("ПГАТУ Инфраструктура")

	mShow := systray.AddMenuItem("Развернуть", "Показать окно")
	mHide := systray.AddMenuItem("Свернуть", "Скрыть окно")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выйти", "Завершить сеанс")
	mLogout := systray.AddMenuItem("Закрыть сессию (КРАШИТ ПРИЛОЖЕНИЕ)", "Выйти из приложения")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				fyne.Do(func() {
					if mainWindow != nil {
						mainWindow.Show()
					} else if loginWindow != nil {
						loginWindow.Show()
					}
				})
			case <-mHide.ClickedCh:
				fyne.Do(func() {
					if mainWindow != nil {
						mainWindow.Hide()
					}
					if loginWindow != nil {
						loginWindow.Hide()
					}
				})
			case <-mLogout.ClickedCh:
				fyne.Do(func() {
					logout()
				})
			case <-mQuit.ClickedCh:
				fyne.Do(func() {
					closeWindowSafely(&mainWindow)
					closeWindowSafely(&loginWindow)
					fyne.CurrentApp().Quit()
					systray.Quit()
				})
				return
			}
		}
	}()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	myApp = app.NewWithID("ru.pgatu.infrastructure")

	var dbErr error
	for i := 0; i < 3; i++ {
		if dbErr = initDB(); dbErr == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if dbErr != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", dbErr)
	}
	defer db.Close()

	go systray.Run(setupTray, func() {
		log.Println("Трей завершил работу")
	})

	if user, err := loadSavedSession(); err == nil {
		currentUser = user
		showMainWindow()
	} else {
		log.Printf("Не удалось загрузить сессию: %v", err)
		showLoginWindow()
	}

	myApp.Run()
}
