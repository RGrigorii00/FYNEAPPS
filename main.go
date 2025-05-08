package main

import (
	tabs "FYNEAPPS/ui"
	"database/sql"
	"fmt"
	"log"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	_ "github.com/lib/pq" // PostgreSQL драйвер
)

var (
	iconData []byte
	db       *sql.DB
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

	// Проверяем соединение
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("ошибка проверки соединения: %v", err)
	}

	log.Println("Успешное подключение к базе данных")
	return nil
}

func main() {
	// Инициализация базы данных
	err := initDB()
	if err != nil {
		log.Printf("Ошибка инициализации БД: %v", err)
		// Приложение может продолжить работу, но без функционала БД
	} else {
		defer db.Close()
	}

	// Создаём новое приложение
	myApp := app.New()

	// Создаем ресурс из встроенных данных
	iconResource := fyne.NewStaticResource("images/icons/pgatu_logo_small.png", iconData)

	// Устанавливаем иконку для приложения
	myApp.SetIcon(iconResource)

	// Создаём главное окно
	window := myApp.NewWindow("ПГАТУ Инфраструктура")
	window.Resize(fyne.NewSize(800, 600))

	// Проверяем поддержку системного трея
	if desk, ok := myApp.(desktop.App); ok && runtime.GOOS == "windows" {
		// Создаем меню для трея
		m := fyne.NewMenu("ПГАТУ Инфраструктура",
			fyne.NewMenuItem("Развернуть", func() {
				window.Show()
			}),
			fyne.NewMenuItem("123", func() {
				window.Show()
			}),
			fyne.NewMenuItem("Выход", func() {
				myApp.Quit()
			}),
		)

		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(iconResource)
	}

	// Обработка сворачивания в трей
	window.SetCloseIntercept(func() {
		window.Hide()
	})

	// Устанавливаем иконку для окна
	window.SetIcon(iconResource)

	// Передаем соединение с БД в создание вкладок
	appTabs := tabs.CreateAppTabs(myApp, window)
	window.SetContent(appTabs)

	// Запускаем приложение
	window.ShowAndRun()
}
