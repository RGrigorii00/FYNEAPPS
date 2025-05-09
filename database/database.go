// Package database предоставляет базовые функции для работы с PostgreSQL
package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq" // Драйвер PostgreSQL
)

// PGConnection представляет соединение с базой данных
type PGConnection struct {
	db *sql.DB
}

// ConnectionOptions параметры для подключения
type ConnectionOptions struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// New создает новое подключение к PostgreSQL
func New() *PGConnection {
	return &PGConnection{}
}

// Connect устанавливает соединение с базой данных
func (pg *PGConnection) Connect(opts ConnectionOptions) error {
	// opts.Host = "83.166.245.249"
	// opts.Port = "5432"
	// opts.User = "user"
	// opts.Password = "user"
	// opts.DBName = "grafana_db"
	// opts.SSLMode = "default"
	if pg.db != nil {
		return fmt.Errorf("соединение уже установлено")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		opts.Host, opts.Port, opts.User, opts.Password, opts.DBName, opts.SSLMode)

	var err error
	pg.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("не удалось подключиться: %v", err)
	}

	// Проверяем соединение
	if err = pg.db.Ping(); err != nil {
		pg.db.Close()
		pg.db = nil
		return fmt.Errorf("проверка соединения не удалась: %v", err)
	}

	log.Println("Успешное подключение к PostgreSQL")
	return nil
}

// Disconnect закрывает соединение с базой данных
func (pg *PGConnection) Disconnect() error {
	if pg.db == nil {
		return fmt.Errorf("соединение не установлено")
	}

	err := pg.db.Close()
	pg.db = nil

	if err != nil {
		return fmt.Errorf("ошибка при закрытии соединения: %v", err)
	}

	log.Println("Соединение с PostgreSQL закрыто")
	return nil
}

// DB возвращает объект базы данных для выполнения запросов
func (pg *PGConnection) DB() *sql.DB {
	return pg.db
}

// IsConnected проверяет активность соединения
func (pg *PGConnection) IsConnected() bool {
	if pg.db == nil {
		return false
	}

	err := pg.db.Ping()
	return err == nil
}
