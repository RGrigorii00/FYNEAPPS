package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type DiskHandler struct {
	DB *sql.DB
}

func NewDiskHandler(db *sql.DB) *DiskHandler {
	return &DiskHandler{DB: db}
}

func (h *DiskHandler) GetDisks(c echo.Context) error {
	rows, err := h.DB.Query("SELECT * FROM disks")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var disks []models.Disk
	for rows.Next() {
		var disk models.Disk
		err := rows.Scan(
			&disk.DiskID,
			&disk.ComputerID,
			&disk.DriveLetter,
			&disk.TotalSpaceGB,
			&disk.UsedSpaceGB,
			&disk.FreeSpaceGB,
			&disk.UsagePercent,
			&disk.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		disks = append(disks, disk)
	}

	return c.JSON(http.StatusOK, disks)
}

func (h *DiskHandler) GetDisk(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var disk models.Disk
	err = h.DB.QueryRow("SELECT * FROM disks WHERE disk_id = $1", id).Scan(
		&disk.DiskID,
		&disk.ComputerID,
		&disk.DriveLetter,
		&disk.TotalSpaceGB,
		&disk.UsedSpaceGB,
		&disk.FreeSpaceGB,
		&disk.UsagePercent,
		&disk.Timestamp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, "Disk not found")
		}
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, disk)
}

func (h *DiskHandler) GetDisksByComputer(c echo.Context) error {
	computerID, err := strconv.Atoi(c.Param("computer_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid Computer ID")
	}

	rows, err := h.DB.Query("SELECT * FROM disks WHERE computer_id = $1", computerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var disks []models.Disk
	for rows.Next() {
		var disk models.Disk
		err := rows.Scan(
			&disk.DiskID,
			&disk.ComputerID,
			&disk.DriveLetter,
			&disk.TotalSpaceGB,
			&disk.UsedSpaceGB,
			&disk.FreeSpaceGB,
			&disk.UsagePercent,
			&disk.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		disks = append(disks, disk)
	}

	return c.JSON(http.StatusOK, disks)
}

func (h *DiskHandler) CreateDisk(c echo.Context) error {
	var disk models.Disk
	if err := c.Bind(&disk); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO disks (
			computer_id, drive_letter, total_space_gb, used_space_gb, 
			free_space_gb, usage_percent
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING disk_id`,
		disk.ComputerID,
		disk.DriveLetter,
		disk.TotalSpaceGB,
		disk.UsedSpaceGB,
		disk.FreeSpaceGB,
		disk.UsagePercent,
	).Scan(&disk.DiskID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, disk)
}

func (h *DiskHandler) UpdateDisk(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var disk models.Disk
	if err := c.Bind(&disk); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err = h.DB.Exec(
		`UPDATE disks SET 
			computer_id = $1, 
			drive_letter = $2, 
			total_space_gb = $3, 
			used_space_gb = $4, 
			free_space_gb = $5, 
			usage_percent = $6
		WHERE disk_id = $7`,
		disk.ComputerID,
		disk.DriveLetter,
		disk.TotalSpaceGB,
		disk.UsedSpaceGB,
		disk.FreeSpaceGB,
		disk.UsagePercent,
		id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	disk.DiskID = id
	return c.JSON(http.StatusOK, disk)
}

func (h *DiskHandler) DeleteDisk(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	_, err = h.DB.Exec("DELETE FROM disks WHERE disk_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
