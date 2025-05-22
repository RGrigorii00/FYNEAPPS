package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type ProcessorHandler struct {
	DB *sql.DB
}

func NewProcessorHandler(db *sql.DB) *ProcessorHandler {
	return &ProcessorHandler{DB: db}
}

func (h *ProcessorHandler) GetProcessors(c echo.Context) error {
	rows, err := h.DB.Query("SELECT * FROM processors")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var processors []models.Processor
	for rows.Next() {
		var processor models.Processor
		err := rows.Scan(
			&processor.ProcessorID,
			&processor.ComputerID,
			&processor.Model,
			&processor.Manufacturer,
			&processor.Architecture,
			&processor.ClockSpeed,
			&processor.CoreCount,
			&processor.ThreadCount,
			&processor.UsagePercent,
			&processor.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		processors = append(processors, processor)
	}

	return c.JSON(http.StatusOK, processors)
}

func (h *ProcessorHandler) GetProcessor(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var processor models.Processor
	err = h.DB.QueryRow("SELECT * FROM processors WHERE processor_id = $1", id).Scan(
		&processor.ProcessorID,
		&processor.ComputerID,
		&processor.Model,
		&processor.Manufacturer,
		&processor.Architecture,
		&processor.ClockSpeed,
		&processor.CoreCount,
		&processor.ThreadCount,
		&processor.UsagePercent,
		&processor.Timestamp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, "Processor not found")
		}
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, processor)
}

func (h *ProcessorHandler) GetProcessorsByComputer(c echo.Context) error {
	computerID, err := strconv.Atoi(c.Param("computer_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid Computer ID")
	}

	rows, err := h.DB.Query("SELECT * FROM processors WHERE computer_id = $1", computerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var processors []models.Processor
	for rows.Next() {
		var processor models.Processor
		err := rows.Scan(
			&processor.ProcessorID,
			&processor.ComputerID,
			&processor.Model,
			&processor.Manufacturer,
			&processor.Architecture,
			&processor.ClockSpeed,
			&processor.CoreCount,
			&processor.ThreadCount,
			&processor.UsagePercent,
			&processor.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		processors = append(processors, processor)
	}

	return c.JSON(http.StatusOK, processors)
}

func (h *ProcessorHandler) CreateProcessor(c echo.Context) error {
	var processor models.Processor
	if err := c.Bind(&processor); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO processors (
			computer_id, model, manufacturer, architecture, clock_speed, 
			core_count, thread_count, usage_percent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING processor_id`,
		processor.ComputerID,
		processor.Model,
		processor.Manufacturer,
		processor.Architecture,
		processor.ClockSpeed,
		processor.CoreCount,
		processor.ThreadCount,
		processor.UsagePercent,
	).Scan(&processor.ProcessorID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, processor)
}

func (h *ProcessorHandler) UpdateProcessor(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var processor models.Processor
	if err := c.Bind(&processor); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err = h.DB.Exec(
		`UPDATE processors SET 
			computer_id = $1, 
			model = $2, 
			manufacturer = $3, 
			architecture = $4, 
			clock_speed = $5, 
			core_count = $6, 
			thread_count = $7, 
			usage_percent = $8
		WHERE processor_id = $9`,
		processor.ComputerID,
		processor.Model,
		processor.Manufacturer,
		processor.Architecture,
		processor.ClockSpeed,
		processor.CoreCount,
		processor.ThreadCount,
		processor.UsagePercent,
		id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	processor.ProcessorID = id
	return c.JSON(http.StatusOK, processor)
}

func (h *ProcessorHandler) DeleteProcessor(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	_, err = h.DB.Exec("DELETE FROM processors WHERE processor_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
