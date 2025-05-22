package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type SoftwareUpdatesHandler struct {
	DB *sql.DB
}

func NewSoftwareUpdatesHandler(db *sql.DB) *SoftwareUpdatesHandler {
	return &SoftwareUpdatesHandler{DB: db}
}

func (h *SoftwareUpdatesHandler) GetUpdates(c echo.Context) error {
	rows, err := h.DB.Query("SELECT * FROM software_updates")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var updates []models.SoftwareUpdate
	for rows.Next() {
		var u models.SoftwareUpdate
		err := rows.Scan(
			&u.UpdateID,
			&u.SoftwareID,
			&u.UpdateName,
			&u.UpdateVersion,
			&u.KBArticle,
			&u.InstallDate,
			&u.SizeMB,
			&u.IsUninstalled,
			&u.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		updates = append(updates, u)
	}
	return c.JSON(http.StatusOK, updates)
}

func (h *SoftwareUpdatesHandler) GetUpdateByID(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var u models.SoftwareUpdate
	err := h.DB.QueryRow("SELECT * FROM software_updates WHERE update_id = $1", id).Scan(
		&u.UpdateID,
		&u.SoftwareID,
		&u.UpdateName,
		&u.UpdateVersion,
		&u.KBArticle,
		&u.InstallDate,
		&u.SizeMB,
		&u.IsUninstalled,
		&u.Timestamp,
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, "Update not found")
	}
	return c.JSON(http.StatusOK, u)
}

func (h *SoftwareUpdatesHandler) CreateUpdate(c echo.Context) error {
	var u models.SoftwareUpdate
	if err := c.Bind(&u); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO software_updates (
			software_id, update_name, update_version, kb_article, 
			install_date, size_mb, is_uninstalled
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING update_id`,
		u.SoftwareID, u.UpdateName, u.UpdateVersion, u.KBArticle,
		u.InstallDate, u.SizeMB, u.IsUninstalled,
	).Scan(&u.UpdateID)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, u)
}

func (h *SoftwareUpdatesHandler) UpdateUpdate(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var u models.SoftwareUpdate
	if err := c.Bind(&u); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err := h.DB.Exec(
		`UPDATE software_updates SET 
			software_id=$1, update_name=$2, update_version=$3, 
			kb_article=$4, install_date=$5, size_mb=$6, 
			is_uninstalled=$7 
		WHERE update_id=$8`,
		u.SoftwareID, u.UpdateName, u.UpdateVersion,
		u.KBArticle, u.InstallDate, u.SizeMB,
		u.IsUninstalled, id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, u)
}

func (h *SoftwareUpdatesHandler) DeleteUpdate(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	_, err := h.DB.Exec("DELETE FROM software_updates WHERE update_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SoftwareUpdatesHandler) GetUpdatesBySoftware(c echo.Context) error {
	softwareID, _ := strconv.Atoi(c.Param("software_id"))
	rows, err := h.DB.Query("SELECT * FROM software_updates WHERE software_id = $1", softwareID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var updates []models.SoftwareUpdate
	for rows.Next() {
		var u models.SoftwareUpdate
		err := rows.Scan(
			&u.UpdateID,
			&u.SoftwareID,
			&u.UpdateName,
			&u.UpdateVersion,
			&u.KBArticle,
			&u.InstallDate,
			&u.SizeMB,
			&u.IsUninstalled,
			&u.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		updates = append(updates, u)
	}
	return c.JSON(http.StatusOK, updates)
}
