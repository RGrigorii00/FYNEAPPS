package models

import "time"

type Computer struct {
	ComputerID     int       `json:"computer_id"`
	HostName       string    `json:"host_name"`
	UserName       string    `json:"user_name"`
	OsName         string    `json:"os_name"`
	OsVersion      string    `json:"os_version"`
	OsPlatform     string    `json:"os_platform"`
	OsArchitecture string    `json:"os_architecture"`
	KernelVersion  string    `json:"kernel_version"`
	Uptime         time.Time `json:"uptime"`
	ProcessCount   int       `json:"process_count"`
	BootTime       time.Time `json:"boot_time"`
	HomeDirectory  string    `json:"home_directory"`
	Gid            string    `json:"gid"`
	Uid            string    `json:"uid"`
}

type Processor struct {
	ProcessorID  int       `json:"processor_id"`
	ComputerID   int       `json:"computer_id"`
	Model        string    `json:"model"`
	Manufacturer string    `json:"manufacturer"`
	Architecture string    `json:"architecture"`
	ClockSpeed   float64   `json:"clock_speed"`
	CoreCount    int       `json:"core_count"`
	ThreadCount  int       `json:"thread_count"`
	UsagePercent float64   `json:"usage_percent"`
	Timestamp    time.Time `json:"timestamp"`
}

type Memory struct {
	MemoryID      int       `json:"memory_id"`
	ComputerID    int       `json:"computer_id"`
	TotalMemoryGB float64   `json:"total_memory_gb"`
	UsedMemoryGB  float64   `json:"used_memory_gb"`
	FreeMemoryGB  float64   `json:"free_memory_gb"`
	UsagePercent  float64   `json:"usage_percent"`
	MemoryType    string    `json:"memory_type"`
	Timestamp     time.Time `json:"timestamp"`
}

type NetworkAdapter struct {
	AdapterID       int       `json:"adapter_id"`
	ComputerID      int       `json:"computer_id"`
	AdapterName     string    `json:"adapter_name"`
	MacAddress      string    `json:"mac_address"`
	UploadSpeed     float64   `json:"upload_speed_mbps"`
	DownloadSpeed   float64   `json:"download_speed_mbps"`
	SentMB          float64   `json:"sent_mb"`
	ReceivedMB      float64   `json:"received_mb"`
	SentPackets     int64     `json:"sent_packets"`
	ReceivedPackets int64     `json:"received_packets"`
	IsActive        bool      `json:"is_active"`
	Timestamp       time.Time `json:"timestamp"`
}

type Disk struct {
	DiskID       int       `json:"disk_id"`
	ComputerID   int       `json:"computer_id"`
	DriveLetter  string    `json:"drive_letter"`
	TotalSpaceGB float64   `json:"total_space_gb"`
	UsedSpaceGB  float64   `json:"used_space_gb"`
	FreeSpaceGB  float64   `json:"free_space_gb"`
	UsagePercent float64   `json:"usage_percent"`
	Timestamp    time.Time `json:"timestamp"`
}

type ComputerSoftware struct {
	ComputerSoftwareID int       `json:"computer_software_id"`
	ComputerID         int       `json:"computer_id"`
	SoftwareID         int       `json:"software_id"`
	IsInstalled        bool      `json:"is_installed"`
	InstallDate        time.Time `json:"install_date"`
	UninstallDate      time.Time `json:"uninstall_date"`
	LastUsed           time.Time `json:"last_used"`
	UsageFrequency     string    `json:"usage_frequency"`
	IsRequired         bool      `json:"is_required"`
	Notes              string    `json:"notes"`
	Timestamp          time.Time `json:"timestamp"`
}

type Software struct {
	SoftwareID        int       `json:"software_id"`
	ComputerID        int       `json:"computer_id"`
	Name              string    `json:"name"`
	Version           string    `json:"version"`
	Publisher         string    `json:"publisher"`
	InstallDate       time.Time `json:"install_date"`
	InstallLocation   string    `json:"install_location"`
	SizeMB            float64   `json:"size_mb"`
	IsSystemComponent bool      `json:"is_system_component"`
	IsUpdate          bool      `json:"is_update"`
	Architecture      string    `json:"architecture"`
	LastUsedDate      time.Time `json:"last_used_date"`
	Timestamp         time.Time `json:"timestamp"`
}

type SoftwareUpdate struct {
	UpdateID      int       `json:"update_id"`
	SoftwareID    int       `json:"software_id"`
	UpdateName    string    `json:"update_name"`
	UpdateVersion string    `json:"update_version"`
	KBArticle     string    `json:"kb_article"`
	InstallDate   time.Time `json:"install_date"`
	SizeMB        float64   `json:"size_mb"`
	IsUninstalled bool      `json:"is_uninstalled"`
	Timestamp     time.Time `json:"timestamp"`
}

type SoftwareDependency struct {
	DependencyID       int       `json:"dependency_id"`
	SoftwareID         int       `json:"software_id"`
	RequiredSoftwareID int       `json:"required_software_id"`
	MinVersion         string    `json:"min_version"`
	MaxVersion         string    `json:"max_version"`
	IsOptional         bool      `json:"is_optional"`
	Timestamp          time.Time `json:"timestamp"`
}
