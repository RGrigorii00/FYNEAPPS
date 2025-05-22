package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type SoftwareDependenciesHandler struct {
	DB *sql.DB
}

func NewSoftwareDependenciesHandler(db *sql.DB) *SoftwareDependenciesHandler {
	return &SoftwareDependenciesHandler{DB: db}
}

func (h *SoftwareDependenciesHandler) GetDependencies(c echo.Context) error {
	rows, err := h.DB.Query(`
		SELECT d.*, s.name as required_software_name 
		FROM software_dependencies d
		JOIN software s ON d.required_software_id = s.software_id
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var deps []struct {
		models.SoftwareDependency
		RequiredSoftwareName string `json:"required_software_name"`
	}

	for rows.Next() {
		var d struct {
			models.SoftwareDependency
			RequiredSoftwareName string `json:"required_software_name"`
		}
		err := rows.Scan(
			&d.DependencyID,
			&d.SoftwareID,
			&d.RequiredSoftwareID,
			&d.MinVersion,
			&d.MaxVersion,
			&d.IsOptional,
			&d.Timestamp,
			&d.RequiredSoftwareName,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		deps = append(deps, d)
	}
	return c.JSON(http.StatusOK, deps)
}

func (h *SoftwareDependenciesHandler) GetDependencyByID(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var d models.SoftwareDependency
	err := h.DB.QueryRow(`
		SELECT d.*, s.name as required_software_name 
		FROM software_dependencies d
		JOIN software s ON d.required_software_id = s.software_id
		WHERE d.dependency_id = $1
	`, id).Scan(
		&d.DependencyID,
		&d.SoftwareID,
		&d.RequiredSoftwareID,
		&d.MinVersion,
		&d.MaxVersion,
		&d.IsOptional,
		&d.Timestamp,
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, "Dependency not found")
	}
	return c.JSON(http.StatusOK, d)
}

func (h *SoftwareDependenciesHandler) CreateDependency(c echo.Context) error {
	var d models.SoftwareDependency
	if err := c.Bind(&d); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO software_dependencies (
			software_id, required_software_id, min_version, 
			max_version, is_optional
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING dependency_id`,
		d.SoftwareID, d.RequiredSoftwareID, d.MinVersion,
		d.MaxVersion, d.IsOptional,
	).Scan(&d.DependencyID)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, d)
}

func (h *SoftwareDependenciesHandler) UpdateDependency(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var d models.SoftwareDependency
	if err := c.Bind(&d); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err := h.DB.Exec(
		`UPDATE software_dependencies SET 
			software_id=$1, required_software_id=$2, 
			min_version=$3, max_version=$4, is_optional=$5 
		WHERE dependency_id=$6`,
		d.SoftwareID, d.RequiredSoftwareID,
		d.MinVersion, d.MaxVersion, d.IsOptional,
		id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, d)
}

func (h *SoftwareDependenciesHandler) DeleteDependency(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	_, err := h.DB.Exec("DELETE FROM software_dependencies WHERE dependency_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SoftwareDependenciesHandler) GetDependenciesBySoftware(c echo.Context) error {
	softwareID, _ := strconv.Atoi(c.Param("software_id"))
	rows, err := h.DB.Query(`
		SELECT d.*, s.name as required_software_name 
		FROM software_dependencies d
		JOIN software s ON d.required_software_id = s.software_id
		WHERE d.software_id = $1
	`, softwareID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var deps []struct {
		models.SoftwareDependency
		RequiredSoftwareName string `json:"required_software_name"`
	}

	for rows.Next() {
		var d struct {
			models.SoftwareDependency
			RequiredSoftwareName string `json:"required_software_name"`
		}
		err := rows.Scan(
			&d.DependencyID,
			&d.SoftwareID,
			&d.RequiredSoftwareID,
			&d.MinVersion,
			&d.MaxVersion,
			&d.IsOptional,
			&d.Timestamp,
			&d.RequiredSoftwareName,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		deps = append(deps, d)
	}
	return c.JSON(http.StatusOK, deps)
}
