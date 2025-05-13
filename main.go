package main

import (
	"FYNEAPPS/resources"
	"FYNEAPPS/ui"
	setting "FYNEAPPS/ui/setting_tab"
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/getlantern/systray"
	_ "github.com/lib/pq"
)

var (
	db     *sql.DB
	myApp  fyne.App
	window fyne.Window
)

const (
	dbHost     = "83.166.245.249"
	dbPort     = 5432
	dbUser     = "user"
	dbPassword = "user"
	dbName     = "grafana_db"
)

func initDB() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	// Устанавливаем разумные лимиты соединений
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("ошибка проверки соединения: %v", err)
	}

	log.Println("Успешное подключение к базе данных")
	return nil
}

func setupTray() {
	// Устанавливаем иконку
	iconData := resources.ResourcePgatulogosmallicoIco.StaticContent
	if len(iconData) > 0 {
		systray.SetIcon(iconData)
	} else {
		log.Println("Предупреждение: данные иконки пустые")
	}

	systray.SetTooltip("ПГАТУ Инфраструктура")

	mShow := systray.AddMenuItem("Развернуть", "Показать окно приложения")
	mHide := systray.AddMenuItem("Свернуть", "Скрыть окно приложения")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Завершить работу приложения")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				fyne.Do(func() {
					window.Show()
					window.RequestFocus()
				})
			case <-mHide.ClickedCh:
				fyne.Do(func() {
					window.Hide()
				})
			case <-mQuit.ClickedCh:
				fyne.Do(func() {
					window.Close()
					myApp.Quit()
					systray.Quit()
				})
				return
			}
		}
	}()
}

func main() {
	// Устанавливаем GOMAXPROCS для лучшей многопоточности
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Инициализация базы данных с повторными попытками
	var dbErr error
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		dbErr = initDB()
		if dbErr == nil {
			break
		}
		log.Printf("Попытка %d: Ошибка инициализации БД: %v", i+1, dbErr)
		time.Sleep(2 * time.Second)
	}

	if dbErr != nil {
		log.Printf("Не удалось подключиться к БД после %d попыток: %v", maxRetries, dbErr)
	} else {
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("Ошибка закрытия соединения с БД: %v", err)
			}
		}()
	}

	myApp = app.NewWithID("ru.pgatu.infrastructure")
	window = myApp.NewWindow("ПГАТУ Инфраструктура")

	// Устанавливаем иконку окна
	if icon := resources.ResourcePgatulogosmallPng; icon != nil {
		window.SetIcon(icon)
	}

	// Загрузка настроек
	appSettings := setting.LoadSettings(myApp, window)
	window.Resize(fyne.NewSize(float32(appSettings.Width), float32(appSettings.Height)))

	// Запускаем кастомный трей
	go systray.Run(setupTray, func() {
		log.Println("Трей завершил работу")
	})

	// Обработка закрытия окна
	window.SetCloseIntercept(func() {
		window.Hide()
	})

	// Создание интерфейса
	appTabs := ui.CreateAppTabs(myApp, window)
	window.SetContent(appTabs)

	// Фоновая задача с защитой от паники
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Восстановлено после паники в фоновой задаче: %v", r)
			}
		}()

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("Выполнение фоновой задачи...")
			fyne.CurrentApp().SendNotification(fyne.NewNotification("ПГАТУ", "Приложение работает в фоне"))
		}
	}()

	// Запуск приложения
	window.ShowAndRun()
}
