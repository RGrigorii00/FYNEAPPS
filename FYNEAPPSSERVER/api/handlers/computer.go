package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type ComputerHandler struct {
	DB *sql.DB
}

func NewComputerHandler(db *sql.DB) *ComputerHandler {
	return &ComputerHandler{DB: db}
}

func (h *ComputerHandler) GetComputers(c echo.Context) error {
	rows, err := h.DB.Query("SELECT * FROM computers")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var computers []models.Computer
	for rows.Next() {
		var computer models.Computer
		err := rows.Scan(
			&computer.ComputerID,
			&computer.HostName,
			&computer.UserName,
			&computer.OsName,
			&computer.OsVersion,
			&computer.OsPlatform,
			&computer.OsArchitecture,
			&computer.KernelVersion,
			&computer.Uptime,
			&computer.ProcessCount,
			&computer.BootTime,
			&computer.HomeDirectory,
			&computer.Gid,
			&computer.Uid,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		computers = append(computers, computer)
	}

	return c.JSON(http.StatusOK, computers)
}

func (h *ComputerHandler) GetComputer(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var computer models.Computer
	err = h.DB.QueryRow("SELECT * FROM computers WHERE computer_id = $1", id).Scan(
		&computer.ComputerID,
		&computer.HostName,
		&computer.UserName,
		&computer.OsName,
		&computer.OsVersion,
		&computer.OsPlatform,
		&computer.OsArchitecture,
		&computer.KernelVersion,
		&computer.Uptime,
		&computer.ProcessCount,
		&computer.BootTime,
		&computer.HomeDirectory,
		&computer.Gid,
		&computer.Uid,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, "Computer not found")
		}
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, computer)
}

func (h *ComputerHandler) CreateComputer(c echo.Context) error {
	var computer models.Computer
	if err := c.Bind(&computer); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO computers (
			host_name, user_name, os_name, os_version, os_platform, 
			os_architecture, kernel_version, uptime, process_count, 
			boot_time, home_directory, gid, uid
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING computer_id`,
		computer.HostName,
		computer.UserName,
		computer.OsName,
		computer.OsVersion,
		computer.OsPlatform,
		computer.OsArchitecture,
		computer.KernelVersion,
		computer.Uptime,
		computer.ProcessCount,
		computer.BootTime,
		computer.HomeDirectory,
		computer.Gid,
		computer.Uid,
	).Scan(&computer.ComputerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, computer)
}

func (h *ComputerHandler) UpdateComputer(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var computer models.Computer
	if err := c.Bind(&computer); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err = h.DB.Exec(
		`UPDATE computers SET 
			host_name = $1, 
			user_name = $2, 
			os_name = $3, 
			os_version = $4, 
			os_platform = $5, 
			os_architecture = $6, 
			kernel_version = $7, 
			uptime = $8, 
			process_count = $9, 
			boot_time = $10, 
			home_directory = $11, 
			gid = $12, 
			uid = $13
		WHERE computer_id = $14`,
		computer.HostName,
		computer.UserName,
		computer.OsName,
		computer.OsVersion,
		computer.OsPlatform,
		computer.OsArchitecture,
		computer.KernelVersion,
		computer.Uptime,
		computer.ProcessCount,
		computer.BootTime,
		computer.HomeDirectory,
		computer.Gid,
		computer.Uid,
		id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	computer.ComputerID = id
	return c.JSON(http.StatusOK, computer)
}

func (h *ComputerHandler) DeleteComputer(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	_, err = h.DB.Exec("DELETE FROM computers WHERE computer_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
