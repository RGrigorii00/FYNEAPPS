package main

import (
	"FYNEAPPS/resources"
	"FYNEAPPS/ui"
	setting "FYNEAPPS/ui/setting_tab"
	"database/sql"
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	_ "github.com/lib/pq"
)

var (
	db *sql.DB
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

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("ошибка проверки соединения: %v", err)
	}

	log.Println("Успешное подключение к базе данных")
	return nil
}

func main() {
	// Инициализация базы данных
	if err := initDB(); err != nil {
		log.Printf("Ошибка инициализации БД: %v", err)
	} else {
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("Ошибка закрытия соединения с БД: %v", err)
			}
		}()
	}

	// Создание приложения
	myApp := app.NewWithID("ru.pgatu.infrastructure")
	iconResource := resources.ResourcePgatulogosmallPng

	// Настройки приложения
	myApp.SetIcon(iconResource)
	window := myApp.NewWindow("ПГАТУ Инфраструктура")

	// Загрузка настроек
	appSettings := setting.LoadSettings(myApp, window)
	window.Resize(fyne.NewSize(float32(appSettings.Width), float32(appSettings.Height)))

	// Настройка системного трея
	if desk, ok := myApp.(desktop.App); ok {
		m := fyne.NewMenu("ПГАТУ Инфраструктура",
			fyne.NewMenuItem("Развернуть", window.Show),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Выход", myApp.Quit),
		)
		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(iconResource)
	}

	// Обработка закрытия окна
	window.SetCloseIntercept(func() {
		window.Hide()
	})

	// Создание интерфейса
	appTabs := ui.CreateAppTabs(myApp, window)
	window.SetContent(appTabs)

	// Фоновая задача (пример)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("Выполнение фоновой задачи...")
			myApp.SendNotification(fyne.NewNotification("ПГАТУ", "Приложение работает в фоне"))
		}
	}()

	// Запуск приложения
	window.ShowAndRun()
}
