package handlers

import (
	"FYNEAPPSSERVER/api/models"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type NetworkHandler struct {
	DB *sql.DB
}

func NewNetworkHandler(db *sql.DB) *NetworkHandler {
	return &NetworkHandler{DB: db}
}

func (h *NetworkHandler) GetNetworkAdapters(c echo.Context) error {
	rows, err := h.DB.Query("SELECT * FROM network_adapters")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var adapters []models.NetworkAdapter
	for rows.Next() {
		var adapter models.NetworkAdapter
		err := rows.Scan(
			&adapter.AdapterID,
			&adapter.ComputerID,
			&adapter.AdapterName,
			&adapter.MacAddress,
			&adapter.UploadSpeed,
			&adapter.DownloadSpeed,
			&adapter.SentMB,
			&adapter.ReceivedMB,
			&adapter.SentPackets,
			&adapter.ReceivedPackets,
			&adapter.IsActive,
			&adapter.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		adapters = append(adapters, adapter)
	}

	return c.JSON(http.StatusOK, adapters)
}

func (h *NetworkHandler) GetNetworkAdapter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var adapter models.NetworkAdapter
	err = h.DB.QueryRow("SELECT * FROM network_adapters WHERE adapter_id = $1", id).Scan(
		&adapter.AdapterID,
		&adapter.ComputerID,
		&adapter.AdapterName,
		&adapter.MacAddress,
		&adapter.UploadSpeed,
		&adapter.DownloadSpeed,
		&adapter.SentMB,
		&adapter.ReceivedMB,
		&adapter.SentPackets,
		&adapter.ReceivedPackets,
		&adapter.IsActive,
		&adapter.Timestamp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, "Network adapter not found")
		}
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, adapter)
}

func (h *NetworkHandler) GetNetworkAdaptersByComputer(c echo.Context) error {
	computerID, err := strconv.Atoi(c.Param("computer_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid Computer ID")
	}

	rows, err := h.DB.Query("SELECT * FROM network_adapters WHERE computer_id = $1", computerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var adapters []models.NetworkAdapter
	for rows.Next() {
		var adapter models.NetworkAdapter
		err := rows.Scan(
			&adapter.AdapterID,
			&adapter.ComputerID,
			&adapter.AdapterName,
			&adapter.MacAddress,
			&adapter.UploadSpeed,
			&adapter.DownloadSpeed,
			&adapter.SentMB,
			&adapter.ReceivedMB,
			&adapter.SentPackets,
			&adapter.ReceivedPackets,
			&adapter.IsActive,
			&adapter.Timestamp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		adapters = append(adapters, adapter)
	}

	return c.JSON(http.StatusOK, adapters)
}

func (h *NetworkHandler) CreateNetworkAdapter(c echo.Context) error {
	var adapter models.NetworkAdapter
	if err := c.Bind(&adapter); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := h.DB.QueryRow(
		`INSERT INTO network_adapters (
			computer_id, adapter_name, mac_address, upload_speed_mbps, 
			download_speed_mbps, sent_mb, received_mb, sent_packets, 
			received_packets, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING adapter_id`,
		adapter.ComputerID,
		adapter.AdapterName,
		adapter.MacAddress,
		adapter.UploadSpeed,
		adapter.DownloadSpeed,
		adapter.SentMB,
		adapter.ReceivedMB,
		adapter.SentPackets,
		adapter.ReceivedPackets,
		adapter.IsActive,
	).Scan(&adapter.AdapterID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, adapter)
}

func (h *NetworkHandler) UpdateNetworkAdapter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	var adapter models.NetworkAdapter
	if err := c.Bind(&adapter); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	_, err = h.DB.Exec(
		`UPDATE network_adapters SET 
			computer_id = $1, 
			adapter_name = $2, 
			mac_address = $3, 
			upload_speed_mbps = $4, 
			download_speed_mbps = $5, 
			sent_mb = $6, 
			received_mb = $7, 
			sent_packets = $8, 
			received_packets = $9, 
			is_active = $10
		WHERE adapter_id = $11`,
		adapter.ComputerID,
		adapter.AdapterName,
		adapter.MacAddress,
		adapter.UploadSpeed,
		adapter.DownloadSpeed,
		adapter.SentMB,
		adapter.ReceivedMB,
		adapter.SentPackets,
		adapter.ReceivedPackets,
		adapter.IsActive,
		id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	adapter.AdapterID = id
	return c.JSON(http.StatusOK, adapter)
}

func (h *NetworkHandler) DeleteNetworkAdapter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid ID")
	}

	_, err = h.DB.Exec("DELETE FROM network_adapters WHERE adapter_id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
