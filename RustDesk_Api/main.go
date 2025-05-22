package main

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Укажите путь к вашей базе данных
	dbPath := "./db_v2.sqlite3"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		panic("Файл базы данных не найден: " + dbPath)
	}

	// Подключение к базе
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	e := echo.New()

	// Обработчик для получения всех записей из таблицы
	e.GET("/data", func(c echo.Context) error {
		// Выполняем простой запрос SELECT *
		rows, err := db.Query("SELECT * FROM peer")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Ошибка запроса: " + err.Error(),
			})
		}
		defer rows.Close()

		// Получаем названия колонок
		columns, err := rows.Columns()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Ошибка получения структуры таблицы",
			})
		}

		// Собираем результаты
		var results []map[string]interface{}
		for rows.Next() {
			// Подготовка значений для сканирования
			values := make([]interface{}, len(columns))
			pointers := make([]interface{}, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}

			// Сканируем строку
			if err := rows.Scan(pointers...); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Ошибка чтения данных",
				})
			}

			// Формируем map с данными
			entry := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				entry[col] = val
			}
			results = append(results, entry)
		}

		return c.JSON(http.StatusOK, results)
	})

	e.Logger.Fatal(e.Start(":8081"))
}
