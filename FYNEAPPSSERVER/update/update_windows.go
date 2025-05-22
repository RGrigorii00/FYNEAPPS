//go:build windows

package update_app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func ApplyUpdate(exePath, downloadPath string) error {
	// Получаем абсолютные пути
	exePath, _ = filepath.Abs(exePath)
	updatePath, _ = filepath.Abs(downloadPath)

	// 1. Находим cmd.exe по абсолютному пути
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = "C:\\Windows"
	}
	cmdPath := filepath.Join(systemRoot, "System32", "cmd.exe")

	// 2. Создаем bat-файл с логикой обновления
	batContent := fmt.Sprintf(`@echo off
:: Основной код обновления
echo [UPDATE] Завершаем текущий процесс...
taskkill /F /IM "%s" >nul 2>&1
ping -n 3 127.0.0.1 >nul

echo [UPDATE] Удаляем старую версию...
if exist "%s" (
    del /F /Q "%s" >nul 2>&1
    if exist "%s" (
        exit /B 1
    )
)

echo [UPDATE] Устанавливаем новую версию...
move /Y "%s" "%s" >nul 2>&1
if not exist "%s" (
    exit /B 2
)

echo [UPDATE] Запускаем обновленную версию...
start "" "%s"
exit /B 0
`, filepath.Base(exePath), exePath, exePath, exePath, updatePath, exePath, exePath, exePath)

	// 3. Сохраняем bat-файл
	batPath := filepath.Join(filepath.Dir(exePath), "update_"+filepath.Base(exePath)+".bat")
	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("ошибка создания bat-файла: %w", err)
	}

	// 4. Запускаем через абсолютный путь к cmd.exe
	if err := runCommandHidden(cmdPath, "/C", batPath); err != nil {
		return fmt.Errorf("ошибка запуска обновления: %w", err)
	}

	return nil
}

func runCommandHidden(path string, args ...string) error {
	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow: true,
	}
	return cmd.Start()
}
