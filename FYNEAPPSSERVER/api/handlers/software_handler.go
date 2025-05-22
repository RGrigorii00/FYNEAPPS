package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type SoftwareHandler struct {
	DB *sql.DB
}

func NewSoftwareHandler(db *sql.DB) *SoftwareHandler {
	return &SoftwareHandler{DB: db}
}

func (h *SoftwareHandler) GetSoftware(c echo.Context) error {
	rows, err := h.DB.Query("SELECT * FROM software")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var software []models.Software
	for rows.Next() {
		var s models.Software
		err := rows.Scan(
			&s.SoftwareID,
			&s.ComputerID,
			&s.Name,
			&s.Version,
			&s.Publisher,
			&s.InstallDate,
			&s.InstallLocation,
			&s.SizeMB,
			&s.IsSystemComponent,
			&s.IsUpdate,
			&s.Architecture,
			&s.LastUsedDate,
			&s.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		software = append(software, s)
	}
	return c.JSON(http.StatusOK, software)
}

func (h *SoftwareHandler) GetSoftwareByID(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var s models.Software
	err := h.DB.QueryRow("SELECT * FROM software WHERE software_id = $1", id).Scan(
		&s.SoftwareID,
		&s.ComputerID,
		&s.Name,
		&s.Version,
		&s.Publisher,
		&s.InstallDate,
		&s.InstallLocation,
		&s.SizeMB,
		&s.IsSystemComponent,
		&s.IsUpdate,
		&s.Architecture,
		&s.LastUsedDate,
		&s.Timestamp,
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, "Software not found")
	}
	return c.JSON(http.StatusOK, s)
}

func (h *SoftwareHandler) CreateSoftware(c echo.Context) error {
	var s models.Software
	if err := c.Bind(&s); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO software (
			computer_id, name, version, publisher, install_date, 
			install_location, size_mb, is_system_component, 
			is_update, architecture, last_used_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING software_id`,
		s.ComputerID, s.Name, s.Version, s.Publisher, s.InstallDate,
		s.InstallLocation, s.SizeMB, s.IsSystemComponent,
		s.IsUpdate, s.Architecture, s.LastUsedDate,
	).Scan(&s.SoftwareID)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, s)
}

func (h *SoftwareHandler) UpdateSoftware(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var s models.Software
	if err := c.Bind(&s); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err := h.DB.Exec(
		`UPDATE software SET 
			computer_id=$1, name=$2, version=$3, publisher=$4, 
			install_date=$5, install_location=$6, size_mb=$7, 
			is_system_component=$8, is_update=$9, architecture=$10, 
			last_used_date=$11 
		WHERE software_id=$12`,
		s.ComputerID, s.Name, s.Version, s.Publisher,
		s.InstallDate, s.InstallLocation, s.SizeMB,
		s.IsSystemComponent, s.IsUpdate, s.Architecture,
		s.LastUsedDate, id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, s)
}

func (h *SoftwareHandler) DeleteSoftware(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	_, err := h.DB.Exec("DELETE FROM software WHERE software_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SoftwareHandler) GetSoftwareByComputer(c echo.Context) error {
	computerID, _ := strconv.Atoi(c.Param("computer_id"))
	rows, err := h.DB.Query("SELECT * FROM software WHERE computer_id = $1", computerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var software []models.Software
	for rows.Next() {
		var s models.Software
		err := rows.Scan(
			&s.SoftwareID,
			&s.ComputerID,
			&s.Name,
			&s.Version,
			&s.Publisher,
			&s.InstallDate,
			&s.InstallLocation,
			&s.SizeMB,
			&s.IsSystemComponent,
			&s.IsUpdate,
			&s.Architecture,
			&s.LastUsedDate,
			&s.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		software = append(software, s)
	}
	return c.JSON(http.StatusOK, software)
}
