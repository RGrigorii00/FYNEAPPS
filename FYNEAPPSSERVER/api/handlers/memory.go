package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type MemoryHandler struct {
	DB *sql.DB
}

func NewMemoryHandler(db *sql.DB) *MemoryHandler {
	return &MemoryHandler{DB: db}
}

func (h *MemoryHandler) GetMemory(c echo.Context) error {
	rows, err := h.DB.Query("SELECT * FROM memory")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var memoryEntries []models.Memory
	for rows.Next() {
		var memory models.Memory
		err := rows.Scan(
			&memory.MemoryID,
			&memory.ComputerID,
			&memory.TotalMemoryGB,
			&memory.UsedMemoryGB,
			&memory.FreeMemoryGB,
			&memory.UsagePercent,
			&memory.MemoryType,
			&memory.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		memoryEntries = append(memoryEntries, memory)
	}

	return c.JSON(http.StatusOK, memoryEntries)
}

func (h *MemoryHandler) GetMemoryEntry(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var memory models.Memory
	err = h.DB.QueryRow("SELECT * FROM memory WHERE memory_id = $1", id).Scan(
		&memory.MemoryID,
		&memory.ComputerID,
		&memory.TotalMemoryGB,
		&memory.UsedMemoryGB,
		&memory.FreeMemoryGB,
		&memory.UsagePercent,
		&memory.MemoryType,
		&memory.Timestamp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, "Memory entry not found")
		}
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, memory)
}

func (h *MemoryHandler) GetMemoryByComputer(c echo.Context) error {
	computerID, err := strconv.Atoi(c.Param("computer_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid Computer ID")
	}

	rows, err := h.DB.Query("SELECT * FROM memory WHERE computer_id = $1", computerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var memoryEntries []models.Memory
	for rows.Next() {
		var memory models.Memory
		err := rows.Scan(
			&memory.MemoryID,
			&memory.ComputerID,
			&memory.TotalMemoryGB,
			&memory.UsedMemoryGB,
			&memory.FreeMemoryGB,
			&memory.UsagePercent,
			&memory.MemoryType,
			&memory.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		memoryEntries = append(memoryEntries, memory)
	}

	return c.JSON(http.StatusOK, memoryEntries)
}

func (h *MemoryHandler) CreateMemoryEntry(c echo.Context) error {
	var memory models.Memory
	if err := c.Bind(&memory); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO memory (
			computer_id, total_memory_gb, used_memory_gb, free_memory_gb, 
			usage_percent, memory_type
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING memory_id`,
		memory.ComputerID,
		memory.TotalMemoryGB,
		memory.UsedMemoryGB,
		memory.FreeMemoryGB,
		memory.UsagePercent,
		memory.MemoryType,
	).Scan(&memory.MemoryID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, memory)
}

func (h *MemoryHandler) UpdateMemoryEntry(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var memory models.Memory
	if err := c.Bind(&memory); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err = h.DB.Exec(
		`UPDATE memory SET 
			computer_id = $1, 
			total_memory_gb = $2, 
			used_memory_gb = $3, 
			free_memory_gb = $4, 
			usage_percent = $5, 
			memory_type = $6
		WHERE memory_id = $7`,
		memory.ComputerID,
		memory.TotalMemoryGB,
		memory.UsedMemoryGB,
		memory.FreeMemoryGB,
		memory.UsagePercent,
		memory.MemoryType,
		id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	memory.MemoryID = id
	return c.JSON(http.StatusOK, memory)
}

func (h *MemoryHandler) DeleteMemoryEntry(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	_, err = h.DB.Exec("DELETE FROM memory WHERE memory_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
