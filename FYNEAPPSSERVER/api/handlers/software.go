package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *SoftwareHandler) GetComputerSoftware(c echo.Context) error {
	computerID, err := strconv.Atoi(c.Param("computer_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid Computer ID")
	}

	rows, err := h.DB.Query("SELECT * FROM computer_software WHERE computer_id = $1", computerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var softwareList []models.ComputerSoftware
	for rows.Next() {
		var software models.ComputerSoftware
		err := rows.Scan(
			&software.ComputerSoftwareID,
			&software.ComputerID,
			&software.SoftwareID,
			&software.IsInstalled,
			&software.InstallDate,
			&software.UninstallDate,
			&software.LastUsed,
			&software.UsageFrequency,
			&software.IsRequired,
			&software.Notes,
			&software.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		softwareList = append(softwareList, software)
	}

	return c.JSON(http.StatusOK, softwareList)
}

func (h *SoftwareHandler) GetSoftwareEntry(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var software models.ComputerSoftware
	err = h.DB.QueryRow("SELECT * FROM computer_software WHERE computer_software_id = $1", id).Scan(
		&software.ComputerSoftwareID,
		&software.ComputerID,
		&software.SoftwareID,
		&software.IsInstalled,
		&software.InstallDate,
		&software.UninstallDate,
		&software.LastUsed,
		&software.UsageFrequency,
		&software.IsRequired,
		&software.Notes,
		&software.Timestamp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, "Software entry not found")
		}
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, software)
}

func (h *SoftwareHandler) CreateSoftwareEntry(c echo.Context) error {
	var software models.ComputerSoftware
	if err := c.Bind(&software); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO computer_software (
			computer_id, software_id, is_installed, install_date, 
			uninstall_date, last_used, usage_frequency, is_required, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING computer_software_id`,
		software.ComputerID,
		software.SoftwareID,
		software.IsInstalled,
		software.InstallDate,
		software.UninstallDate,
		software.LastUsed,
		software.UsageFrequency,
		software.IsRequired,
		software.Notes,
	).Scan(&software.ComputerSoftwareID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, software)
}

func (h *SoftwareHandler) UpdateSoftwareEntry(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var software models.ComputerSoftware
	if err := c.Bind(&software); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err = h.DB.Exec(
		`UPDATE computer_software SET 
			computer_id = $1, 
			software_id = $2, 
			is_installed = $3, 
			install_date = $4, 
			uninstall_date = $5, 
			last_used = $6, 
			usage_frequency = $7, 
			is_required = $8, 
			notes = $9
		WHERE computer_software_id = $10`,
		software.ComputerID,
		software.SoftwareID,
		software.IsInstalled,
		software.InstallDate,
		software.UninstallDate,
		software.LastUsed,
		software.UsageFrequency,
		software.IsRequired,
		software.Notes,
		id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	software.ComputerSoftwareID = id
	return c.JSON(http.StatusOK, software)
}

func (h *SoftwareHandler) DeleteSoftwareEntry(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	_, err = h.DB.Exec("DELETE FROM computer_software WHERE computer_software_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
