// server/main.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

const (
	uploadDirName = "uploads" // Имя папки для загрузок (будет создана в корне)
	port          = ":10051"
)

func main() {
	e := echo.New()

	// Жёстко задаём путь к bin директории
	binPath := "/usr/local/bin"

	// Полный путь к папке загрузок
	uploadDir := filepath.Join(binPath, uploadDirName)

	// Проверяем существование bin директории
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		e.Logger.Fatalf("Директория %s не существует", binPath)
	}

	// Создаем папку для файлов если не существует
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		e.Logger.Fatalf("Не удалось создать папку uploads: %v", err)
	}

	// Устанавливаем правильные права (на случай если папка уже существовала)
	if err := os.Chmod(uploadDir, 0755); err != nil {
		e.Logger.Warnf("Не удалось изменить права на папку: %v", err)
	}

	e.Logger.Printf("Папка для загрузок: %s", uploadDir)

	// Роуты (без изменений)
	e.GET("/download", func(c echo.Context) error {
		return downloadHandler(c, uploadDir)
	})
	e.GET("/files", func(c echo.Context) error {
		return listFilesHandler(c, uploadDir)
	})

	e.Logger.Printf("Сервер запущен на порту %s", port)
	e.Logger.Fatal(e.Start(port))
}

// getRootDir возвращает абсолютный путь к корневой директории приложения
func getRootDir() (string, error) {
	// Получаем путь к исполняемому файлу
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Переходим в родительскую директорию (если exe в bin/)
	rootDir := filepath.Dir(filepath.Dir(exePath))

	// Проверяем, есть ли в этой директории папка bin (опционально)
	if _, err := os.Stat(filepath.Join(rootDir, "bin")); err == nil {
		return rootDir, nil
	}

	// Если папки bin нет, возвращаем директорию с исполняемым файлом
	return filepath.Dir(exePath), nil
}

func downloadHandler(c echo.Context, uploadDir string) error {
	// Получаем имя файла из параметра запроса
	filename := c.QueryParam("file")
	if filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Не указано имя файла")
	}

	// Безопасное получение пути к файлу
	filePath := filepath.Join(uploadDir, filepath.Base(filename))

	// Проверяем существование файла
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return echo.NewHTTPError(http.StatusNotFound, "Файл не найден")
	}

	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Ошибка при открытии файла")
	}
	defer file.Close()

	// Устанавливаем заголовки
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
	c.Response().Header().Set(echo.HeaderContentLength, fmt.Sprintf("%d", fileInfo.Size()))

	// Отправляем файл
	return c.Stream(http.StatusOK, "application/octet-stream", file)
}

func listFilesHandler(c echo.Context, uploadDir string) error {
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Ошибка чтения директории")
	}

	var fileList []string
	for _, file := range files {
		if !file.IsDir() {
			fileList = append(fileList, file.Name())
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"files": fileList,
	})
}
