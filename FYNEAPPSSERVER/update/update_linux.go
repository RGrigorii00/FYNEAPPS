//go:build !windows

package update_app

import (
	"fmt"
	"os"
)

func applyUpdate(exePath, downloadPath string) error {
	// Устанавливаем права на новый файл
	if err := os.Chmod(downloadPath, 0755); err != nil {
		return fmt.Errorf("ошибка установки прав: %w", err)
	}

	// Заменяем файл
	if err := os.Rename(downloadPath, exePath); err != nil {
		return fmt.Errorf("ошибка замены файла: %w", err)
	}

	return nil
}
