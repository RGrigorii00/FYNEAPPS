CREATE TABLE software (
    software_id SERIAL PRIMARY KEY,
    computer_id INTEGER REFERENCES computers(computer_id),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(100),
    publisher VARCHAR(255),
    install_date DATE,
    install_location VARCHAR(512),
    size_mb DECIMAL(10,2),
    is_system_component BOOLEAN DEFAULT FALSE,
    is_update BOOLEAN DEFAULT FALSE,
    architecture VARCHAR(20), -- x86, x64, ARM и т.д.
    last_used_date TIMESTAMP,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE software_updates (
    update_id SERIAL PRIMARY KEY,
    software_id INTEGER REFERENCES software(software_id),
    update_name VARCHAR(255),
    update_version VARCHAR(100),
    kb_article VARCHAR(50), -- для Windows-обновлений
    install_date TIMESTAMP,
    size_mb DECIMAL(10,2),
    is_uninstalled BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE software_dependencies (
    dependency_id SERIAL PRIMARY KEY,
    software_id INTEGER REFERENCES software(software_id),
    required_software_id INTEGER REFERENCES software(software_id),
    min_version VARCHAR(100),
    max_version VARCHAR(100),
    is_optional BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Вставка данных в таблицу software
INSERT INTO software (computer_id, name, version, publisher, install_date, install_location, size_mb, is_system_component, is_update, architecture, last_used_date)
VALUES
    (1, 'Microsoft Office', '2019', 'Microsoft Corporation', '2022-01-15', 'C:\Program Files\Microsoft Office', 2500.50, FALSE, FALSE, 'x64', '2023-05-10 14:30:00'),
    (1, 'Adobe Photoshop', '23.1.1', 'Adobe Inc.', '2022-03-10', 'C:\Program Files\Adobe\Photoshop', 3500.75, FALSE, FALSE, 'x64', '2023-05-08 10:15:00'),
    (1, 'Google Chrome', '112.0.5615.138', 'Google LLC', '2023-04-01', 'C:\Program Files (x86)\Google\Chrome', 450.25, FALSE, FALSE, 'x86', '2023-05-10 09:00:00'),
    (1, 'Visual Studio Code', '1.77.3', 'Microsoft Corporation', '2023-03-20', 'C:\Users\User\AppData\Local\Programs\Microsoft VS Code', 500.00, FALSE, FALSE, 'x64', '2023-05-09 16:45:00'),
    (1, 'Windows Defender', '4.18.2203.5', 'Microsoft Corporation', '2021-11-15', 'C:\Program Files\Windows Defender', 1200.00, TRUE, FALSE, 'x64', '2023-05-10 08:30:00');

-- Вставка данных в таблицу software_updates
INSERT INTO software_updates (software_id, update_name, update_version, kb_article, install_date, size_mb, is_uninstalled)
VALUES
    (11, 'Office Security Update', '2019.1', 'KB5002100', '2023-02-15 10:00:00', 150.25, FALSE),
    (12, 'Office Feature Update', '2019.2', 'KB5002101', '2023-03-20 11:30:00', 250.50, FALSE),
    (13, 'Photoshop Bug Fix', '23.1.2', NULL, '2023-04-05 09:15:00', 75.75, FALSE),
    (14, 'Chrome Security Update', '112.0.5615.139', NULL, '2023-04-15 14:00:00', 30.00, FALSE),
    (15, 'Defender Definitions', '4.18.2204.1', 'KB5002102', '2023-04-01 03:00:00', 80.00, FALSE);

-- Вставка данных в таблицу software_dependencies
INSERT INTO software_dependencies (software_id, required_software_id, min_version, max_version, is_optional)
VALUES
    (11, 11, '110.0.0.0', NULL, FALSE),  -- Office требует Chrome версии не ниже 110
    (12, 11, '2016.0.0.0', NULL, TRUE),  -- Photoshop может использовать Office (опционально)
    (13, 11, '100.0.0.0', NULL, FALSE),  -- VS Code требует Chrome
    (14, NULL, NULL, NULL, FALSE),      -- Defender не имеет зависимостей
    (15, NULL, NULL, NULL, FALSE);       -- Chrome не имеет зависимостей

ALTER TABLE software 
ADD COLUMN download_url VARCHAR(1024);

SELECT 
    name AS "Название ПО",
    publisher AS "Производитель",
    COUNT(*) AS "Количество установок",
    ROUND(AVG(size_mb), 2) AS "Средний размер (MB)",
    MAX(version) AS "Последняя версия",
    COUNT(DISTINCT architecture) AS "Поддерживаемые архитектуры"
FROM 
    software
WHERE 
    is_system_component = FALSE 
    AND is_update = FALSE
GROUP BY 
    name, publisher
ORDER BY 
    COUNT(*) DESC