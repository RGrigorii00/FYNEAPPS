CREATE TABLE computers (
	computer_id SERIAL PRIMARY KEY,
    host_name VARCHAR(100) NOT NULL,
    user_name VARCHAR(100),
    os_name VARCHAR(50) NOT NULL,
    os_version VARCHAR(100) NOT NULL,
    os_platform VARCHAR(50),
    os_architecture VARCHAR(20),
    kernel_version VARCHAR(100),
    uptime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    process_count INTEGER,
    boot_time TIMESTAMP,
    home_directory VARCHAR(255),
    gid VARCHAR(100),
    uid VARCHAR(100)
);

CREATE TABLE processors (
    processor_id SERIAL PRIMARY KEY,
    computer_id INTEGER REFERENCES computers(computer_id),
    model VARCHAR(100) NOT NULL,
    manufacturer VARCHAR(100),
    architecture VARCHAR(20),
    clock_speed DECIMAL(8,2), -- in GHz
    core_count INTEGER,
    thread_count INTEGER,
    usage_percent DECIMAL(5,2),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE memory (
    memory_id SERIAL PRIMARY KEY,
    computer_id INTEGER REFERENCES computers(computer_id),
    total_memory_gb DECIMAL(8,2),
    used_memory_gb DECIMAL(8,2),
    free_memory_gb DECIMAL(8,2),
    usage_percent DECIMAL(5,2),
    memory_type VARCHAR(50),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE network_adapters (
    adapter_id SERIAL PRIMARY KEY,
    computer_id INTEGER REFERENCES computers(computer_id),
    adapter_name VARCHAR(100) NOT NULL,
    mac_address VARCHAR(50),
    upload_speed_mbps DECIMAL(10,2),
    download_speed_mbps DECIMAL(10,2),
    sent_mb DECIMAL(12,2),
    received_mb DECIMAL(12,2),
    sent_packets BIGINT,
    received_packets BIGINT,
    is_active BOOLEAN DEFAULT TRUE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE disks (
    disk_id SERIAL PRIMARY KEY,
    computer_id INTEGER REFERENCES computers(computer_id),
    drive_letter VARCHAR(5) NOT NULL,
    total_space_gb DECIMAL(12,2),
    used_space_gb DECIMAL(12,2),
    free_space_gb DECIMAL(12,2),
    usage_percent DECIMAL(5,2),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE computer_software (
    computer_software_id SERIAL PRIMARY KEY,
    computer_id INTEGER NOT NULL REFERENCES computers(computer_id),
    software_id INTEGER NOT NULL REFERENCES software(software_id),
    is_installed BOOLEAN DEFAULT TRUE,
    install_date TIMESTAMP,
    uninstall_date TIMESTAMP,
    last_used TIMESTAMP,
    usage_frequency VARCHAR(50), -- например: "daily", "weekly", "monthly", "rarely"
    is_required BOOLEAN DEFAULT FALSE, -- обязательно ли это ПО для данного компьютера
    notes TEXT, -- дополнительные заметки
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (computer_id, software_id) -- чтобы избежать дублирования записей
);